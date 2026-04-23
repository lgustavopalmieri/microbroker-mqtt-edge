package domain

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	c := NewClient("test-client", server, 60)

	assert.Equal(t, "test-client", c.ID)
	assert.Equal(t, uint16(60), c.KeepAlive)
	assert.NotNil(t, c.Conn)
	assert.False(t, c.CreatedAt.IsZero())
	assert.False(t, c.LastSeen.IsZero())
}

func TestClient_Write(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	c := NewClient("writer", server, 60)

	go func() {
		buf := make([]byte, 5)
		n, _ := client.Read(buf)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("hello"), buf[:n])
	}()

	err := c.Write([]byte("hello"))
	require.NoError(t, err)
}

func TestClient_Close(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	c := NewClient("closer", server, 60)
	err := c.Close()
	require.NoError(t, err)

	// Writing to closed conn should fail
	err = c.Write([]byte("data"))
	assert.Error(t, err)
}

func TestClient_ResetDeadline(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	c := NewClient("deadline", server, 10)
	before := c.LastSeen

	time.Sleep(5 * time.Millisecond)
	c.ResetDeadline()

	assert.True(t, c.LastSeen.After(before))
}

func TestClient_ResetDeadline_ZeroKeepAlive(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// KeepAlive=0 means no deadline
	c := NewClient("no-deadline", server, 0)
	c.ResetDeadline() // should not panic or set deadline
}
