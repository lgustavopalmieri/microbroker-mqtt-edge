package topic

import (
	"fmt"
	"strings"
	"sync"
)

// TopicRegistry holds the set of allowed topics and tracks dynamic ownership.
// Each topic can be owned by at most one client at a time.
// A client may own multiple topics simultaneously.
// Thread-safe for concurrent use from multiple connection goroutines.
type TopicRegistry struct {
	mu      sync.RWMutex
	allowed map[string]struct{}
	owners  map[string]string // topic → clientID
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

	return &TopicRegistry{
		allowed: allowed,
		owners:  make(map[string]string, len(topics)),
	}, nil
}

// IsAllowed returns true if the topic is in the allowed set.
func (r *TopicRegistry) IsAllowed(topic string) bool {
	_, ok := r.allowed[topic]
	return ok
}

// Claim attempts to assign ownership of a topic to a client.
// Returns nil if:
//   - the topic has no owner (client becomes owner), or
//   - the topic is already owned by the same client.
//
// Returns ErrTopicNotAllowed if the topic is not in the allowed set.
// Returns ErrTopicOwnedByAnother if a different client already owns the topic.
func (r *TopicRegistry) Claim(topic, clientID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.allowed[topic]; !ok {
		return ErrTopicNotAllowed
	}

	owner, exists := r.owners[topic]
	if exists && owner != clientID {
		return fmt.Errorf("%w: topic %q owned by %q", ErrTopicOwnedByAnother, topic, owner)
	}

	r.owners[topic] = clientID
	return nil
}

// Release removes ownership of all topics held by the given client.
// Called when a client disconnects.
func (r *TopicRegistry) Release(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for topic, owner := range r.owners {
		if owner == clientID {
			delete(r.owners, topic)
		}
	}
}

// Owner returns the clientID that owns the given topic, or empty string if unowned.
func (r *TopicRegistry) Owner(topic string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.owners[topic]
}

// Topics returns all allowed topics as a slice.
func (r *TopicRegistry) Topics() []string {
	topics := make([]string, 0, len(r.allowed))
	for t := range r.allowed {
		topics = append(topics, t)
	}
	return topics
}
