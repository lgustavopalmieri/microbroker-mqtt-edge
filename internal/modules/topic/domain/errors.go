package domain

import "errors"

var (
	// ErrInvalidTopicCount indicates the topic count is outside the allowed range (1-5).
	ErrInvalidTopicCount = errors.New("topic: topic count must be between 1 and 5")

	// ErrEmptyTopicName indicates an empty topic name was provided.
	ErrEmptyTopicName = errors.New("topic: empty topic name")
)
