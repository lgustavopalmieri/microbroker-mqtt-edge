package ingestion

import "errors"

var (
	// ErrStoreFailure indicates a persistence operation failed.
	ErrStoreFailure = errors.New("ingestion: store failure")
)
