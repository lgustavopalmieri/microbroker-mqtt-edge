package http_handler

import "microbroker-mqtt-edge/internal/modules/oee/availability/features/query/application"

// Handler exposes the availability query endpoint.
type Handler struct {
	useCase *application.UseCase
}

// NewHandler creates a Handler backed by the given use case.
func NewHandler(uc *application.UseCase) *Handler {
	return &Handler{useCase: uc}
}
