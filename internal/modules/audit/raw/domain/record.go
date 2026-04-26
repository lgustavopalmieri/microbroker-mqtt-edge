package domain

// Record represents a single persisted message returned by audit queries.
type Record struct {
	ClientID  string `json:"client_id"`
	Topic     string `json:"topic"`
	Timezone  string `json:"timezone"`
	Timestamp string `json:"timestamp"`
	Payload   string `json:"payload"`
}
