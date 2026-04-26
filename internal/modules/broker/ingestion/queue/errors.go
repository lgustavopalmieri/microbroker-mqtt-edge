package queue

import "errors"

var (
	// ErrQueueFull indicates the topic queue buffer is full (backpressure).
	ErrQueueFull = errors.New("ingestion: queue full")
)
