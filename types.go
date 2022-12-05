package workerpool

import (
	"sync"
	"time"
)

type Handler[T any] func(T)

type Config struct {
	MaxWorkersCount       uint
	MaxIdleWorkerDuration time.Duration
}

type Pool[T any] struct {
	cfg     Config
	handler Handler[T]

	mu           sync.Mutex
	workersCount uint
	mustStop     bool
	ready        []*workerChan[T]
	stopCh       chan struct{}

	workerChanPool sync.Pool
}

type workerChan[T any] struct {
	lastUseTime time.Time
	dataCh      chan T
	stopCh      chan struct{}
}
