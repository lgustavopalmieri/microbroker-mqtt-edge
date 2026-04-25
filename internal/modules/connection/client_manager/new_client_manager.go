package clientmanager

import (
	"sync"

	"microbroker-mqtt-edge/internal/modules/connection/client"
)

// ClientManager tracks active client connections with a configurable limit.
type ClientManager struct {
	mu         sync.Mutex
	clients    map[string]*client.Client
	maxClients int
}

// NewClientManager creates a ClientManager with the given max client limit.
func NewClientManager(maxClients int) *ClientManager {
	return &ClientManager{
		clients:    make(map[string]*client.Client),
		maxClients: maxClients,
	}
}
