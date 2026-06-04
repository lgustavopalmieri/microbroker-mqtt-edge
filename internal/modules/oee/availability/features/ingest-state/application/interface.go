package application

import (
	"context"
	"time"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

// IntervalStore is the outbound port for persisting state intervals.
type IntervalStore interface {
	OpenInterval(ctx context.Context, machineID string, state ooedomain.MachineState, startedAt time.Time) error
	CloseOpen(ctx context.Context, machineID string, endedAt time.Time) error
	LastOpen(ctx context.Context, machineID string) (avdomain.StateInterval, bool, error)
}

// StateObserver is notified after a valid state transition is persisted.
type StateObserver interface {
	Apply(t avdomain.StateTransition)
}

// Logger is the observability contract used by this use case.
type Logger = observability.Logger
