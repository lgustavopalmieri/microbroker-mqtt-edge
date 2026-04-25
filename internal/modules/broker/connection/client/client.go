package client

import "time"

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
