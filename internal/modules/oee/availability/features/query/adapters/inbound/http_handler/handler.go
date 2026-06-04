package http_handler

import (
	"encoding/json"
	"net/http"
	"time"

	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

// RegisterRoutes registers GET /availability/{machine} on mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /availability/{machine}", h.getAvailability)
}

func (h *Handler) getAvailability(w http.ResponseWriter, r *http.Request) {
	machineID := r.PathValue("machine")
	q := r.URL.Query()

	fromStr := q.Get("from")
	toStr := q.Get("to")
	if fromStr == "" || toStr == "" {
		http.Error(w, `{"error":"from and to are required"}`, http.StatusBadRequest)
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		http.Error(w, `{"error":"invalid from timestamp"}`, http.StatusBadRequest)
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		http.Error(w, `{"error":"invalid to timestamp"}`, http.StatusBadRequest)
		return
	}

	if !from.Before(to) {
		http.Error(w, `{"error":"from must be before to"}`, http.StatusBadRequest)
		return
	}

	output, err := h.useCase.Execute(r.Context(), machineID, ooedomain.Window{From: from, To: to})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(output)
}
