package domain

import "errors"

var (
	// ErrQueryFailed indicates a read query against the audit store failed.
	ErrQueryFailed = errors.New("audit: query failed")
)
