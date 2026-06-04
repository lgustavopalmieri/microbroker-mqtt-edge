package http_handler

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// RegisterRoutes registers GET /availability/{machine}/ws on mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /availability/{machine}/ws", h.serveWS)
}

func (h *Handler) serveWS(w http.ResponseWriter, r *http.Request) {
	machineID := r.PathValue("machine")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("ws handler: upgrade failed", "error", err, "machine_id", machineID)
		return
	}
	h.hub.Register(machineID, conn)
	go h.readLoop(machineID, conn)
}

// readLoop discards inbound frames and unregisters the conn on any error
// (including normal close). Required by gorilla/websocket to process control
// frames such as ping/pong/close.
func (h *Handler) readLoop(machineID string, conn *websocket.Conn) {
	defer h.hub.Unregister(machineID, conn)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
