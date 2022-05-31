# workerpool

[![Test status](https://github.com/savsgio/workerpool/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/savsgio/workerpool/actions?workflow=test)
[![Go Report Card](https://goreportcard.com/badge/github.com/savsgio/workerpool)](https://goreportcard.com/report/github.com/savsgio/workerpool)
[![GoDev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/savsgio/workerpool)

Lightweight and fast worker pool with generics support.

Go version: >=1.18

Based on: [valyala/fasthttp/workerpool.go](https://github.com/valyala/fasthttp)

## Example:

```go
type Foo struct {
    A int
    B int
}

wg := sync.WaitGroup{}

handler := func(args Foo) {
    log.Println(args.A + args.B)
    wg.Done()
}

wp := New(
    Config{
        MaxWorkersCount:       100,
        MaxIdleWorkerDuration: 5 * time.Second,
    },
    handler,
)

for i := 0; i < 10; i++ {
    wg.Add(1)
    wp.Exec(Foo{A: i, B: i + 1})
}

wg.Wait()
```
