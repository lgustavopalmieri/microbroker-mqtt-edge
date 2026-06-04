package logsink

import (
	"context"
	"sync"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
)

// LogSink is an AvailabilitySink that logs each snapshot at debug level and
// retains the latest snapshot per machine for the future /live endpoint.
type LogSink struct {
	logger observability.Logger
	latest sync.Map // map[string]avdomain.AvailabilitySnapshot
}

// NewLogSink creates a LogSink backed by the given logger.
func NewLogSink(logger observability.Logger) *LogSink {
	return &LogSink{logger: logger}
}

// Update logs the snapshot and stores it as the latest for the machine.
func (s *LogSink) Update(_ context.Context, snap avdomain.AvailabilitySnapshot) error {
	s.logger.Debug("availability snapshot",
		"machine_id", snap.MachineID,
		"availability", snap.Availability,
		"has_data", snap.HasData,
		"computed_at", snap.ComputedAt,
	)
	s.latest.Store(snap.MachineID, snap)
	return nil
}

// Latest returns the most recently stored snapshot for the given machine.
// Returns found=false if no snapshot has been received for that machine.
func (s *LogSink) Latest(machineID string) (avdomain.AvailabilitySnapshot, bool) {
	v, ok := s.latest.Load(machineID)
	if !ok {
		return avdomain.AvailabilitySnapshot{}, false
	}
	return v.(avdomain.AvailabilitySnapshot), true
}
