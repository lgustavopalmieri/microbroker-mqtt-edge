package domain

import (
	"net"
	"sync"
	"time"
)

// Client represents an active MQTT client connection.
type Client struct {
	ID        string
	Conn      net.Conn
	KeepAlive uint16
	CreatedAt time.Time
	LastSeen  time.Time
	mu        sync.Mutex
}

// NewClient creates a new Client with the given connection metadata.
func NewClient(id string, conn net.Conn, keepAlive uint16) *Client {
	now := time.Now()
	return &Client{
		ID:        id,
		Conn:      conn,
		KeepAlive: keepAlive,
		CreatedAt: now,
		LastSeen:  now,
	}
}

// ResetDeadline updates the TCP connection deadline to 1.5x the keep-alive interval.
// If keep-alive is 0, no deadline is set (infinite).
func (c *Client) ResetDeadline() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastSeen = time.Now()
	if c.KeepAlive > 0 {
		timeout := time.Duration(float64(c.KeepAlive)*1.5) * time.Second
		c.Conn.SetDeadline(time.Now().Add(timeout))
	}
}

// Write sends data to the client's TCP connection.
func (c *Client) Write(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.Conn.Write(data)
	return err
}

// Close closes the client's TCP connection.
func (c *Client) Close() error {
	return c.Conn.Close()
}
