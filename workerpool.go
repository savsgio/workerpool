package workerpool

import (
	"runtime"
	"sync"
	"time"

	gtime "github.com/savsgio/gotils/time"
)

var workerChanCap = func() int {
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}

	return 1
}()

func New[T any](cfg Config, handler Handler[T]) *Pool[T] {
	if cfg.MaxWorkersCount <= 0 {
		cfg.MaxWorkersCount = defaultMaxWorkersCount
	}

	if cfg.MaxIdleWorkerDuration <= 0 {
		cfg.MaxIdleWorkerDuration = defaultMaxIdleWorkerDuration
	}

	wp := &Pool[T]{
		cfg:     cfg,
		handler: handler,
		stopCh:  make(chan struct{}),
		ready:   make([]*workerChan[T], 0, cfg.MaxWorkersCount),
		workerChanPool: sync.Pool{
			New: func() interface{} {
				return &workerChan[T]{
					dataCh: make(chan T, workerChanCap),
					stopCh: make(chan struct{}, workerChanCap),
				}
			},
		},
	}

	wp.startGC()

	return wp
}

func (wp *Pool[T]) startGC() {
	stopCh := wp.stopCh

	go func() {
		ticker := time.NewTicker(wp.cfg.MaxIdleWorkerDuration)
		defer ticker.Stop()

		stop := false

		for !stop {
			select {
			case <-ticker.C:
			case <-stopCh:
				stop = true
			}

			wp.clean()
		}
	}()
}

func (wp *Pool[T]) Stop() {
	close(wp.stopCh)
	wp.stopCh = nil

	// Stop all the workers waiting for incoming handlers.
	// Do not wait for busy workers - they will stop after
	// serving the connection and noticing wp.mustStop = true.
	wp.mu.Lock()

	for i := range wp.ready {
		close(wp.ready[i].stopCh)
		close(wp.ready[i].dataCh)
	}

	wp.ready = wp.ready[:0]
	wp.mustStop = true

	wp.mu.Unlock()
}

func (wp *Pool[T]) indexOf(criticalTime time.Time) int {
	// Use binary-search algorithm to find out the index of the least recently worker which can be cleaned up.
	n := len(wp.ready) - 1
	i, j := 0, n

	for i <= j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h

		if criticalTime.After(wp.ready[h].lastUseTime) {
			i = h + 1
		} else {
			j = h - 1
		}
	}

	return j
}

func (wp *Pool[T]) clean() {
	// Clean least recently used workers if they didn't anything
	// for more than maxIdleWorkerDuration.
	criticalTime := time.Now().Add(-wp.cfg.MaxIdleWorkerDuration)

	wp.mu.Lock()

	idx := wp.indexOf(criticalTime)
	if idx == -1 {
		wp.mu.Unlock()

		return
	}

	idx++
	chToStop := wp.ready[:idx]
	wp.ready = wp.ready[idx:]

	wp.mu.Unlock()

	// Notify obsolete workers to stop.
	// This notification must be outside the wp.mu, since ch.ch
	// may be blocking and may consume a lot of time if many workers
	// are located on non-local CPUs.
	for i := range chToStop {
		chToStop[i].stopCh <- struct{}{}
	}
}

func (wp *Pool[T]) getCh() *workerChan[T] {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	ready := wp.ready
	if n := len(ready) - 1; n >= 0 {
		ch := ready[n]
		wp.ready = ready[:n]

		return ch
	}

	if wp.workersCount > wp.cfg.MaxWorkersCount {
		return nil
	}

	wp.workersCount++

	vch := wp.workerChanPool.Get()

	ch, ok := vch.(*workerChan[T])
	if !ok {
		panic("unexpected type of workerChan")
	}

	go func() {
		wp.serve(ch)
		wp.workerChanPool.Put(vch)
	}()

	return ch
}

func (wp *Pool[T]) release(ch *workerChan[T]) bool {
	ch.lastUseTime = time.Now()

	wp.mu.Lock()

	if wp.mustStop {
		wp.mu.Unlock()

		return false
	}

	wp.ready = append(wp.ready, ch)

	wp.mu.Unlock()

	return true
}

func (wp *Pool[T]) serve(ch *workerChan[T]) {
	stop := false

	for !stop {
		select {
		case <-ch.stopCh:
			stop = true
		case value := <-ch.dataCh:
			wp.handler(value)

			if !wp.release(ch) {
				break
			}
		}
	}

	wp.mu.Lock()
	wp.workersCount--
	wp.mu.Unlock()
}

func (wp *Pool[T]) Exec(args T) {
	var ticker *time.Ticker

	ok := false

	for !ok {
		ch := wp.getCh()
		if ch == nil {
			if ticker == nil {
				ticker = gtime.AcquireTicker(10 * time.Millisecond)
			}

			// Wait to retry to get a worker channel again.
			<-ticker.C

			continue
		}

		if ticker != nil {
			gtime.ReleaseTicker(ticker)
		}

		ch.dataCh <- args

		ok = true
	}
}
