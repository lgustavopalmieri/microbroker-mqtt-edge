package session

import (
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/session/domain"
)

func newTestClient(id string) (*domain.Client, func()) {
	server, client := net.Pipe()
	c := domain.NewClient(id, server, 60)
	cleanup := func() {
		server.Close()
		client.Close()
	}
	return c, cleanup
}

func TestConnectionManager_AddAndCount(t *testing.T) {
	cm := NewConnectionManager(5)
	assert.Equal(t, 0, cm.Count())

	c1, cleanup := newTestClient("c1")
	defer cleanup()

	err := cm.Add(c1)
	require.NoError(t, err)
	assert.Equal(t, 1, cm.Count())
}

func TestConnectionManager_CanAccept(t *testing.T) {
	cm := NewConnectionManager(2)
	assert.True(t, cm.CanAccept())

	c1, cl1 := newTestClient("c1")
	defer cl1()
	c2, cl2 := newTestClient("c2")
	defer cl2()

	cm.Add(c1)
	assert.True(t, cm.CanAccept())

	cm.Add(c2)
	assert.False(t, cm.CanAccept())
}

func TestConnectionManager_AddBeyondLimit(t *testing.T) {
	cm := NewConnectionManager(1)

	c1, cl1 := newTestClient("c1")
	defer cl1()
	c2, cl2 := newTestClient("c2")
	defer cl2()

	err := cm.Add(c1)
	require.NoError(t, err)

	err = cm.Add(c2)
	assert.ErrorIs(t, err, domain.ErrMaxClientsReached)
}

func TestConnectionManager_Remove(t *testing.T) {
	cm := NewConnectionManager(2)

	c1, cl1 := newTestClient("c1")
	defer cl1()

	cm.Add(c1)
	assert.Equal(t, 1, cm.Count())

	cm.Remove("c1")
	assert.Equal(t, 0, cm.Count())
	assert.True(t, cm.CanAccept())
}

func TestConnectionManager_Get(t *testing.T) {
	cm := NewConnectionManager(5)

	c1, cl1 := newTestClient("c1")
	defer cl1()
	cm.Add(c1)

	got, ok := cm.Get("c1")
	assert.True(t, ok)
	assert.Equal(t, "c1", got.ID)

	_, ok = cm.Get("nonexistent")
	assert.False(t, ok)
}

func TestConnectionManager_ReplaceExisting(t *testing.T) {
	cm := NewConnectionManager(1)

	c1, cl1 := newTestClient("same-id")
	defer cl1()
	c2, cl2 := newTestClient("same-id")
	defer cl2()

	err := cm.Add(c1)
	require.NoError(t, err)

	// Replace with same ID should succeed even at limit
	err = cm.Add(c2)
	require.NoError(t, err)
	assert.Equal(t, 1, cm.Count())
}

func TestConnectionManager_CloseAll(t *testing.T) {
	cm := NewConnectionManager(5)

	c1, cl1 := newTestClient("c1")
	defer cl1()
	c2, cl2 := newTestClient("c2")
	defer cl2()

	cm.Add(c1)
	cm.Add(c2)
	assert.Equal(t, 2, cm.Count())

	cm.CloseAll()
	assert.Equal(t, 0, cm.Count())
}

func TestConnectionManager_ConcurrentAdd(t *testing.T) {
	cm := NewConnectionManager(5)
	var wg sync.WaitGroup
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c, cleanup := newTestClient(string(rune('a' + idx)))
			defer cleanup()
			errors[idx] = cm.Add(c)
		}(i)
	}

	wg.Wait()

	successCount := 0
	failCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		} else {
			failCount++
		}
	}

	assert.Equal(t, 5, successCount)
	assert.Equal(t, 5, failCount)
	assert.Equal(t, 5, cm.Count())
}
