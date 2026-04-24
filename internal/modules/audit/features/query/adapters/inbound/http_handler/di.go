package http_handler

import (
	"microbroker-mqtt-edge/internal/modules/audit/features/query/application"
)

// Handler exposes HTTP endpoints for audit queries.
type Handler struct {
	useCase *application.UseCase
}

// NewHandler creates an HTTP handler backed by the query use case.
func NewHandler(useCase *application.UseCase) *Handler {
	return &Handler{useCase: useCase}
}
