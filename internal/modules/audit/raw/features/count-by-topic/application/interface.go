package application

import (
	"context"

	"microbroker-mqtt-edge/internal/common/observability"
)

// Repository defines the outbound port for counting audit records.
type Repository interface {
	CountByTopic(ctx context.Context, topic string) (int64, error)
}

// Logger is the observability contract used by this use case.
type Logger = observability.Logger
