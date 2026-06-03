package application_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/application"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

func TestDecodeStateChange(t *testing.T) {
	validTS := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		payload     []byte
		expectError bool
		validate    func(t *testing.T, tr application.StateTransition)
	}{
		{
			name: "happy path - valid payload decodes all fields into StateTransition",
			payload: []byte(`{
				"type": "state_change",
				"machine_id": "CNC-01",
				"state": "stopped",
				"previous_state": "running",
				"reason": "breakdown",
				"timestamp": "2024-01-15T09:00:00Z"
			}`),
			validate: func(t *testing.T, tr application.StateTransition) {
				assert.Equal(t, "CNC-01", tr.MachineID)
				assert.Equal(t, ooedomain.Stopped, tr.State)
				assert.Equal(t, ooedomain.Running, tr.PreviousState)
				assert.Equal(t, "breakdown", tr.Reason)
				assert.Equal(t, validTS, tr.Timestamp)
			},
		},
		{
			name: "happy path - optional previous_state and reason are preserved when present",
			payload: []byte(`{
				"type": "state_change",
				"machine_id": "M2",
				"state": "setup",
				"previous_state": "running",
				"reason": "changeover",
				"timestamp": "2024-01-15T09:00:00Z"
			}`),
			validate: func(t *testing.T, tr application.StateTransition) {
				assert.Equal(t, ooedomain.MachineState("running"), tr.PreviousState)
				assert.Equal(t, "changeover", tr.Reason)
			},
		},
		{
			name:        "failure - invalid JSON returns wrapped error",
			payload:     []byte(`not json`),
			expectError: true,
		},
		{
			name:        "failure - missing machine_id returns error",
			payload:     []byte(`{"type":"state_change","machine_id":"","state":"running","timestamp":"2024-01-15T09:00:00Z"}`),
			expectError: true,
		},
		{
			name:        "failure - unknown state value returns error",
			payload:     []byte(`{"type":"state_change","machine_id":"CNC-01","state":"broken","timestamp":"2024-01-15T09:00:00Z"}`),
			expectError: true,
		},
		{
			name:        "failure - zero/missing timestamp returns error",
			payload:     []byte(`{"type":"state_change","machine_id":"CNC-01","state":"running"}`),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr, err := application.DecodeStateChange(tc.payload)

			if tc.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.validate != nil {
				tc.validate(t, tr)
			}
		})
	}
}
