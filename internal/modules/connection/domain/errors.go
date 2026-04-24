package domain

import "errors"

var (
	// ErrMaxClientsReached indicates the broker has reached its maximum client limit.
	ErrMaxClientsReached = errors.New("connection: max clients reached")

	// ErrClientAlreadyExists indicates a client with the same ID is already connected.
	ErrClientAlreadyExists = errors.New("connection: client already exists")

	// ErrConnectionTimeout indicates the client did not send a CONNECT packet in time.
	ErrConnectionTimeout = errors.New("connection: connection timeout")

	// ErrInvalidTopicCount indicates the topic count is outside the allowed range (1-5).
	ErrInvalidTopicCount = errors.New("connection: topic count must be between 1 and 5")

	// ErrEmptyTopicName indicates an empty topic name was provided.
	ErrEmptyTopicName = errors.New("connection: empty topic name")
)
