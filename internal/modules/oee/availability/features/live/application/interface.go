package application

import (
	"context"
	"time"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

// AvailabilitySink is the output port for emitting availability snapshots.
type AvailabilitySink interface {
	Update(ctx context.Context, s avdomain.AvailabilitySnapshot) error
}

// ShiftReader is the outbound port for reading planned production time.
type ShiftReader interface {
	ForMachineWindow(ctx context.Context, machineID string, w ooedomain.Window) (planned time.Duration, found bool, err error)
}

// StateIntervalReader reads the last open interval for a machine (used at startup for rehydration).
type StateIntervalReader interface {
	LastOpen(ctx context.Context, machineID string) (avdomain.StateInterval, bool, error)
}

// Logger is the observability contract used by this engine.
type Logger = observability.Logger
