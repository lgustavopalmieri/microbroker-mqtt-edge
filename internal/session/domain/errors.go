package domain

import "errors"

var (
	// ErrMaxClientsReached indicates the broker has reached its maximum client limit.
	ErrMaxClientsReached = errors.New("session: max clients reached")

	// ErrClientAlreadyExists indicates a client with the same ID is already connected.
	ErrClientAlreadyExists = errors.New("session: client already exists")

	// ErrAuthFailed indicates authentication failed (bad username or password).
	ErrAuthFailed = errors.New("session: authentication failed")

	// ErrConnectionTimeout indicates the client did not send a CONNECT packet in time.
	ErrConnectionTimeout = errors.New("session: connection timeout")

	// ErrInvalidTopicCount indicates the topic count is outside the allowed range (1-5).
	ErrInvalidTopicCount = errors.New("session: topic count must be between 1 and 5")

	// ErrEmptyTopicName indicates an empty topic name was provided.
	ErrEmptyTopicName = errors.New("session: empty topic name")
)
