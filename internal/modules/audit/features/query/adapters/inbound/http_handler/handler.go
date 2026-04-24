package http_handler

import (
	"encoding/json"
	"net/http"

	"microbroker-mqtt-edge/internal/modules/audit/domain"
)

// RegisterRoutes registers the audit query endpoints on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /audit/{topic...}", h.getByTopic)
	mux.HandleFunc("GET /audit-count/{topic...}", h.countByTopic)
}

// getByTopic handles GET /audit/{topic...}
func (h *Handler) getByTopic(w http.ResponseWriter, r *http.Request) {
	topic := r.PathValue("topic")
	if topic == "" {
		http.Error(w, `{"error":"topic is required"}`, http.StatusBadRequest)
		return
	}

	output, err := h.useCase.GetByTopic(r.Context(), topic)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	records := output.Records
	if records == nil {
		records = []domain.Record{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// countByTopic handles GET /audit-count/{topic...}
func (h *Handler) countByTopic(w http.ResponseWriter, r *http.Request) {
	topic := r.PathValue("topic")
	if topic == "" {
		http.Error(w, `{"error":"topic is required"}`, http.StatusBadRequest)
		return
	}

	output, err := h.useCase.CountByTopic(r.Context(), topic)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"count": output.Count})
}
