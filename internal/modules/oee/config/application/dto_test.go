package application_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/modules/oee/config/application"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

func TestLoadShiftsFromJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectError bool
		validate    func(t *testing.T, shifts []ooedomain.Shift)
	}{
		{
			name: "happy path - parses valid JSON array into []Shift with all fields mapped",
			input: []byte(`[{
				"id": "s1",
				"name": "Manhã",
				"machine_id": "CNC-01",
				"start_minute": 480,
				"end_minute": 960,
				"weekdays": [1,2,3,4,5],
				"timezone": "America/Sao_Paulo",
				"breaks": []
			}]`),
			validate: func(t *testing.T, shifts []ooedomain.Shift) {
				require.Len(t, shifts, 1)
				s := shifts[0]
				assert.Equal(t, "s1", s.ID)
				assert.Equal(t, "Manhã", s.Name)
				assert.Equal(t, "CNC-01", s.MachineID)
				assert.Equal(t, 480, s.StartMin)
				assert.Equal(t, 960, s.EndMin)
				assert.Equal(t, "America/Sao_Paulo", s.TZ)
			},
		},
		{
			name: "happy path - weekday integers map to correct time.Weekday values",
			input: []byte(`[{
				"id": "s1",
				"name": "Turno",
				"machine_id": "*",
				"start_minute": 0,
				"end_minute": 480,
				"weekdays": [0,1,6],
				"timezone": "UTC",
				"breaks": []
			}]`),
			validate: func(t *testing.T, shifts []ooedomain.Shift) {
				require.Len(t, shifts, 1)
				assert.Equal(t, []time.Weekday{time.Sunday, time.Monday, time.Saturday}, shifts[0].Weekdays)
			},
		},
		{
			name: "happy path - break fields (start_minute, end_minute, type) map correctly",
			input: []byte(`[{
				"id": "s1",
				"name": "Turno",
				"machine_id": "*",
				"start_minute": 480,
				"end_minute": 960,
				"weekdays": [],
				"timezone": "UTC",
				"breaks": [
					{"start_minute": 600, "end_minute": 615, "type": "rest"},
					{"start_minute": 720, "end_minute": 750, "type": "meal"}
				]
			}]`),
			validate: func(t *testing.T, shifts []ooedomain.Shift) {
				require.Len(t, shifts, 1)
				require.Len(t, shifts[0].Breaks, 2)
				assert.Equal(t, ooedomain.Break{StartMin: 600, EndMin: 615, Type: "rest"}, shifts[0].Breaks[0])
				assert.Equal(t, ooedomain.Break{StartMin: 720, EndMin: 750, Type: "meal"}, shifts[0].Breaks[1])
			},
		},
		{
			name:  "happy path - empty JSON array returns empty slice without error",
			input: []byte(`[]`),
			validate: func(t *testing.T, shifts []ooedomain.Shift) {
				assert.Empty(t, shifts)
			},
		},
		{
			name:        "failure - invalid JSON returns a wrapped error",
			input:       []byte(`not json`),
			expectError: true,
		},
		{
			name:        "failure - JSON object instead of array returns error",
			input:       []byte(`{"id":"s1"}`),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			shifts, err := application.LoadShiftsFromJSON(tc.input)

			if tc.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.validate != nil {
				tc.validate(t, shifts)
			}
		})
	}
}
