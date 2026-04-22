package audit

import (
	"encoding/json"
	"net/http"
)

// Handler exposes a REST endpoint for querying persisted messages by topic.
type Handler struct {
	reader Reader
	logger Logger
}

// NewHandler creates an audit Handler with the given dependencies.
func NewHandler(reader Reader, logger Logger) *Handler {
	return &Handler{reader: reader, logger: logger}
}

// RegisterRoutes registers the audit endpoints on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /audit/{topic...}", h.getByTopic)
}

// getByTopic handles GET /audit/{topic...}
// The topic is extracted from the path wildcard so it supports slashes (e.g. machine/status).
func (h *Handler) getByTopic(w http.ResponseWriter, r *http.Request) {
	topic := r.PathValue("topic")
	if topic == "" {
		http.Error(w, `{"error":"topic is required"}`, http.StatusBadRequest)
		return
	}

	records, err := h.reader.GetByTopic(r.Context(), topic)
	if err != nil {
		h.logger.Error("audit query failed", "topic", topic, "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if records == nil {
		records = []Record{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}
