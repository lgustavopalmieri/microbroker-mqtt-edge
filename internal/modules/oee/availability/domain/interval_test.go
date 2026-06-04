package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

func TestStateInterval_IsOpen(t *testing.T) {
	start := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	tests := []struct {
		name     string
		interval avdomain.StateInterval
		want     bool
	}{
		{
			name:     "nil EndedAt → IsOpen returns true",
			interval: avdomain.StateInterval{State: ooedomain.Running, StartedAt: start, EndedAt: nil},
			want:     true,
		},
		{
			name:     "non-nil EndedAt → IsOpen returns false",
			interval: avdomain.StateInterval{State: ooedomain.Running, StartedAt: start, EndedAt: &end},
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.interval.IsOpen())
		})
	}
}
