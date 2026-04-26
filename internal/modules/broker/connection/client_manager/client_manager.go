package clientmanager

import (
	"microbroker-mqtt-edge/internal/modules/broker/connection/client"
)

// CanAccept returns true if the manager can accept another client.
func (cm *ClientManager) CanAccept() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return len(cm.clients) < cm.maxClients
}

// Add registers a client. Returns ErrMaxClientsReached if the limit is hit.
// If a client with the same ID already exists, the old one is closed and replaced.
func (cm *ClientManager) Add(c *client.Client) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.clients) >= cm.maxClients {
		// Check if we're replacing an existing client (same ID)
		if _, exists := cm.clients[c.ID]; !exists {
			return client.ErrMaxClientsReached
		}
	}

	// Close existing client with same ID if present
	if existing, exists := cm.clients[c.ID]; exists {
		existing.Close()
	}

	cm.clients[c.ID] = c
	return nil
}

// Remove unregisters a client by ID and frees the slot.
func (cm *ClientManager) Remove(clientID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.clients, clientID)
}

// Get returns a client by ID, or false if not found.
func (cm *ClientManager) Get(clientID string) (*client.Client, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	c, ok := cm.clients[clientID]
	return c, ok
}

// Count returns the number of active clients.
func (cm *ClientManager) Count() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return len(cm.clients)
}

// CloseAll closes all active client connections.
func (cm *ClientManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for id, c := range cm.clients {
		c.Close()
		delete(cm.clients, id)
	}
}
