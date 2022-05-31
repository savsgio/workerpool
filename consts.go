package workerpool

import "time"

const (
	defaultMaxWorkersCount       = 256 * 1024
	defaultMaxIdleWorkerDuration = 10 * time.Second
)
