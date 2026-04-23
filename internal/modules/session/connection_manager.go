package session

import (
	"sync"

	"microbroker-mqtt-edge/internal/modules/session/domain"
)

// ConnectionManager tracks active client connections with a configurable limit.
type ConnectionManager struct {
	mu         sync.Mutex
	clients    map[string]*domain.Client
	maxClients int
}

// NewConnectionManager creates a ConnectionManager with the given max client limit.
func NewConnectionManager(maxClients int) *ConnectionManager {
	return &ConnectionManager{
		clients:    make(map[string]*domain.Client),
		maxClients: maxClients,
	}
}

// CanAccept returns true if the manager can accept another client.
func (cm *ConnectionManager) CanAccept() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return len(cm.clients) < cm.maxClients
}

// Add registers a client. Returns ErrMaxClientsReached if the limit is hit.
// If a client with the same ID already exists, the old one is closed and replaced.
func (cm *ConnectionManager) Add(client *domain.Client) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.clients) >= cm.maxClients {
		// Check if we're replacing an existing client (same ID)
		if _, exists := cm.clients[client.ID]; !exists {
			return domain.ErrMaxClientsReached
		}
	}

	// Close existing client with same ID if present
	if existing, exists := cm.clients[client.ID]; exists {
		existing.Close()
	}

	cm.clients[client.ID] = client
	return nil
}

// Remove unregisters a client by ID and frees the slot.
func (cm *ConnectionManager) Remove(clientID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.clients, clientID)
}

// Get returns a client by ID, or false if not found.
func (cm *ConnectionManager) Get(clientID string) (*domain.Client, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	c, ok := cm.clients[clientID]
	return c, ok
}

// Count returns the number of active clients.
func (cm *ConnectionManager) Count() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return len(cm.clients)
}

// CloseAll closes all active client connections.
func (cm *ConnectionManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for id, c := range cm.clients {
		c.Close()
		delete(cm.clients, id)
	}
}
