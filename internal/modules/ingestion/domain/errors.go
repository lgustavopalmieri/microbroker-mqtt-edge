package domain

import "errors"

var (
	// ErrStoreFailure indicates a persistence operation failed.
	ErrStoreFailure = errors.New("ingestion: store failure")

	// ErrQueueFull indicates the topic queue buffer is full (backpressure).
	ErrQueueFull = errors.New("ingestion: queue full")
)
