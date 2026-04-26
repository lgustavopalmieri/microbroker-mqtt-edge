package client

import "errors"

var (
	// ErrMaxClientsReached indicates the broker has reached its maximum client limit.
	ErrMaxClientsReached = errors.New("connection: max clients reached")

	// ErrClientAlreadyExists indicates a client with the same ID is already connected.
	ErrClientAlreadyExists = errors.New("connection: client already exists")

	// ErrConnectionTimeout indicates the client did not send a CONNECT packet in time.
	ErrConnectionTimeout = errors.New("connection: connection timeout")
)
