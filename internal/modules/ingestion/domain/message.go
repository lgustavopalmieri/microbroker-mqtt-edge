package domain

import "time"

// Message represents a data point received from a client via PUBLISH,
// ready to be persisted and forwarded to workers.
type Message struct {
	ClientID  string
	Topic     string
	Payload   []byte
	Timezone  string
	Timestamp time.Time
}
