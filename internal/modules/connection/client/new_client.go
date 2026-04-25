package client

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
