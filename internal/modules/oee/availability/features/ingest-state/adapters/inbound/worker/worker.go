package worker

import (
	"context"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/application"
)

// StateIngester is the inbound port for applying a decoded state transition.
type StateIngester interface {
	Apply(ctx context.Context, t avdomain.StateTransition) error
}

// Logger is the observability contract used by this worker.
type Logger = observability.Logger

// StateChangeWorker is a fanout.Worker that self-filters to the configured
// state topic, decodes state_change payloads, and delegates to a use case.
type StateChangeWorker struct {
	stateTopic string
	ingester   StateIngester
	logger     Logger
}

// NewStateChangeWorker creates a StateChangeWorker with the given state topic,
// use-case ingester, and logger.
func NewStateChangeWorker(stateTopic string, ingester StateIngester, logger Logger) *StateChangeWorker {
	return &StateChangeWorker{
		stateTopic: stateTopic,
		ingester:   ingester,
		logger:     logger,
	}
}

// Name returns "oee-state-change".
func (w *StateChangeWorker) Name() string { return "oee-state-change" }

// Process self-filters on topic, decodes the payload, and delegates to the
// ingester. Non-state messages and malformed payloads are silently skipped.
func (w *StateChangeWorker) Process(ctx context.Context, msg message.Message) error {
	if msg.Topic != w.stateTopic {
		return nil
	}

	transition, err := application.DecodeStateChange(msg.Payload)
	if err != nil {
		w.logger.Warn("oee-state-change: skipping malformed payload", "error", err, "topic", msg.Topic)
		return nil
	}

	if err := w.ingester.Apply(ctx, transition); err != nil {
		w.logger.Error("oee-state-change: apply failed", "error", err, "machine_id", transition.MachineID)
		return err
	}

	return nil
}

// Close is a no-op — this worker holds no resources.
func (w *StateChangeWorker) Close() error { return nil }
