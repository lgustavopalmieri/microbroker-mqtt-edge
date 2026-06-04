package http_handler

import (
	"microbroker-mqtt-edge/internal/common/observability"
	oeeWS "microbroker-mqtt-edge/internal/modules/oee/availability/features/live/adapters/outbound/sink/websocket"
)

// Handler serves the WebSocket upgrade endpoint for live availability.
type Handler struct {
	hub    *oeeWS.Hub
	logger observability.Logger
}

// NewHandler creates a Handler backed by the given hub.
func NewHandler(hub *oeeWS.Hub, logger observability.Logger) *Handler {
	return &Handler{hub: hub, logger: logger}
}
