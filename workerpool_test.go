package workerpool

import (
	"io"
	"log"
	"sync"
	"testing"
)

func Benchmark_Workerpool(b *testing.B) {
	type foo struct {
		A int
		B int
	}

	log := log.New(io.Discard, "", 0)
	wg := sync.WaitGroup{}

	handler := func(args foo) {
		log.Println(args.A + args.B)
		wg.Done()
	}

	wp := New(Config{}, handler)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wg.Add(1)
			wp.Exec(foo{A: 1, B: 2})
		}
	})

	wg.Wait()
}

func Benchmark_Gorutines(b *testing.B) {
	type foo struct {
		A int
		B int
	}

	log := log.New(io.Discard, "", 0)
	wg := sync.WaitGroup{}

	handler := func(args foo) {
		log.Println(args.A + args.B)
		wg.Done()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wg.Add(1)

			go handler(foo{A: 1, B: 2})
		}
	})

	wg.Wait()
}
