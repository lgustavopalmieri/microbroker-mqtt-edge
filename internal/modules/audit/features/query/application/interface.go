package application

import (
	"context"

	"microbroker-mqtt-edge/internal/common/observability"
	"microbroker-mqtt-edge/internal/modules/audit/domain"
)

// Repository defines the outbound port for reading persisted audit records.
// Implementations must be safe for concurrent use.
type Repository interface {
	GetByTopic(ctx context.Context, topic string) ([]domain.Record, error)
	CountByTopic(ctx context.Context, topic string) (int64, error)
}

// Logger is the observability contract used by the query use case.
type Logger = observability.Logger
