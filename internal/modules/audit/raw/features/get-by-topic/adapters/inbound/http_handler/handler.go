package http_handler

import (
	"encoding/json"
	"net/http"

	"microbroker-mqtt-edge/internal/modules/audit/raw/domain"
)

// RegisterRoutes registers the get-by-topic endpoint on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /audit/{topic...}", h.getByTopic)
}

func (h *Handler) getByTopic(w http.ResponseWriter, r *http.Request) {
	topic := r.PathValue("topic")
	if topic == "" {
		http.Error(w, `{"error":"topic is required"}`, http.StatusBadRequest)
		return
	}

	output, err := h.useCase.Execute(r.Context(), topic)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	records := output.Records
	if records == nil {
		records = []domain.Record{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(records)
}
