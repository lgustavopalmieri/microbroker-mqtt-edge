package application

import (
	"context"

	"microbroker-mqtt-edge/internal/common/observability"
	"microbroker-mqtt-edge/internal/modules/audit/domain"
)

// Repository defines the outbound port for reading audit records by topic.
type Repository interface {
	GetByTopic(ctx context.Context, topic string) ([]domain.Record, error)
}

// Logger is the observability contract used by this use case.
type Logger = observability.Logger
