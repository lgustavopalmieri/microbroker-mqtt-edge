package queue

import (
	"context"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
)

// Store defines the persistence contract used by the queue consumer.
type Store interface {
	SaveRawData(ctx context.Context, msg message.Message) error
	Close() error
}

// Logger is the observability contract used by the queue.
type Logger = observability.Logger
