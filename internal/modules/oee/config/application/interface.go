package application

import (
	"context"
	"time"

	"microbroker-mqtt-edge/internal/common/observability"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

// ShiftStore is the outbound port for persisting and querying shift configuration.
type ShiftStore interface {
	Upsert(ctx context.Context, s ooedomain.Shift) error
	ForMachineWindow(ctx context.Context, machineID string, w ooedomain.Window) (planned time.Duration, found bool, err error)
}

// Logger is the observability contract used by this use case.
type Logger = observability.Logger
