package application

import (
	"encoding/json"
	"fmt"
	"time"

	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

type stateChangeJSON struct {
	Type          string    `json:"type"`
	MachineID     string    `json:"machine_id"`
	State         string    `json:"state"`
	PreviousState string    `json:"previous_state"`
	Reason        string    `json:"reason"`
	Timestamp     time.Time `json:"timestamp"`
}

// DecodeStateChange parses and validates a state_change JSON payload.
func DecodeStateChange(payload []byte) (avdomain.StateTransition, error) {
	var raw stateChangeJSON
	if err := json.Unmarshal(payload, &raw); err != nil {
		return avdomain.StateTransition{}, fmt.Errorf("ingest-state: invalid JSON: %w", err)
	}
	if raw.MachineID == "" {
		return avdomain.StateTransition{}, fmt.Errorf("ingest-state: missing machine_id")
	}
	state := ooedomain.MachineState(raw.State)
	if !state.Valid() {
		return avdomain.StateTransition{}, fmt.Errorf("ingest-state: unknown state %q", raw.State)
	}
	if raw.Timestamp.IsZero() {
		return avdomain.StateTransition{}, fmt.Errorf("ingest-state: missing timestamp")
	}
	return avdomain.StateTransition{
		MachineID:     raw.MachineID,
		State:         state,
		PreviousState: ooedomain.MachineState(raw.PreviousState),
		Reason:        raw.Reason,
		Timestamp:     raw.Timestamp,
	}, nil
}
