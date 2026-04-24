package message

import "time"

// Message represents a data point received from an MQTT client via PUBLISH.
// This is the canonical value object shared across module boundaries.
type Message struct {
	ClientID  string
	Topic     string
	Payload   []byte
	Timezone  string
	Timestamp time.Time
}
