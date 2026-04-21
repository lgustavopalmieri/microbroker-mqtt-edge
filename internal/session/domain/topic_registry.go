package domain

import (
	"fmt"
	"strings"
)

// TopicRegistry holds the set of allowed topics for the broker.
type TopicRegistry struct {
	allowed map[string]struct{}
}

// NewTopicRegistry creates a TopicRegistry from the given topic list.
// Validates: 1-5 topics, no empty names.
func NewTopicRegistry(topics []string) (*TopicRegistry, error) {
	if len(topics) == 0 || len(topics) > 5 {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidTopicCount, len(topics))
	}

	allowed := make(map[string]struct{}, len(topics))
	for _, t := range topics {
		t = strings.TrimSpace(t)
		if t == "" {
			return nil, ErrEmptyTopicName
		}
		allowed[t] = struct{}{}
	}

	return &TopicRegistry{allowed: allowed}, nil
}

// IsAllowed returns true if the topic is in the allowed set.
func (r *TopicRegistry) IsAllowed(topic string) bool {
	_, ok := r.allowed[topic]
	return ok
}

// Topics returns all allowed topics as a slice.
func (r *TopicRegistry) Topics() []string {
	topics := make([]string, 0, len(r.allowed))
	for t := range r.allowed {
		topics = append(topics, t)
	}
	return topics
}
