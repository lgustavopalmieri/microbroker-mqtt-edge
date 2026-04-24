package http_handler

import (
	"microbroker-mqtt-edge/internal/modules/audit/features/count-by-topic/application"
)

// Handler exposes the HTTP endpoint for count-by-topic.
type Handler struct {
	useCase *application.UseCase
}

// NewHandler creates an HTTP handler backed by the count-by-topic use case.
func NewHandler(useCase *application.UseCase) *Handler {
	return &Handler{useCase: useCase}
}
