package application

import (
	"context"
	"time"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

// IntervalReader is the outbound port for reading persisted state intervals.
type IntervalReader interface {
	ByMachineRange(ctx context.Context, machineID string, from, to time.Time) ([]avdomain.StateInterval, error)
}

// ShiftReader is the outbound port for reading planned production time.
type ShiftReader interface {
	ForMachineWindow(ctx context.Context, machineID string, w ooedomain.Window) (planned time.Duration, found bool, err error)
}

// Logger is the observability contract used by this use case.
type Logger = observability.Logger
