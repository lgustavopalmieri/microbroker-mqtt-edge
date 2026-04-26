package http_handler

import (
	"encoding/json"
	"net/http"
)

// RegisterRoutes registers the count-by-topic endpoint on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /audit-count/{topic...}", h.countByTopic)
}

func (h *Handler) countByTopic(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int64{"count": output.Count})
}
