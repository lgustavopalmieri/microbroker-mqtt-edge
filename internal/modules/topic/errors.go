package topic

import "errors"

var (
	// ErrInvalidTopicCount indicates the topic count is outside the allowed range (1-5).
	ErrInvalidTopicCount = errors.New("topic: topic count must be between 1 and 5")

	// ErrEmptyTopicName indicates an empty topic name was provided.
	ErrEmptyTopicName = errors.New("topic: empty topic name")

	// ErrTopicOwnedByAnother indicates a client tried to publish to a topic
	// that is already owned by a different client.
	ErrTopicOwnedByAnother = errors.New("topic: owned by another client")

	// ErrTopicNotAllowed indicates the topic is not in the allowed set.
	ErrTopicNotAllowed = errors.New("topic: not allowed")
)
