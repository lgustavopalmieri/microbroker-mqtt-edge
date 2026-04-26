package topic

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTopicRegistry_Valid(t *testing.T) {
	topics := []string{"machine/status", "machine/alarm", "machine/oee"}
	reg, err := NewTopicRegistry(topics)
	require.NoError(t, err)
	assert.Equal(t, 3, len(reg.Topics()))
}

func TestNewTopicRegistry_ZeroTopics(t *testing.T) {
	_, err := NewTopicRegistry([]string{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTopicCount)
}

func TestNewTopicRegistry_SixTopics(t *testing.T) {
	topics := []string{"a", "b", "c", "d", "e", "f"}
	_, err := NewTopicRegistry(topics)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTopicCount)
}

func TestNewTopicRegistry_EmptyTopicName(t *testing.T) {
	_, err := NewTopicRegistry([]string{"valid", ""})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyTopicName)
}

func TestNewTopicRegistry_TrimsSpaces(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"  machine/status  "})
	require.NoError(t, err)
	assert.True(t, reg.IsAllowed("machine/status"))
}

func TestTopicRegistry_IsAllowed(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status", "machine/alarm"})
	require.NoError(t, err)

	assert.True(t, reg.IsAllowed("machine/status"))
	assert.True(t, reg.IsAllowed("machine/alarm"))
	assert.False(t, reg.IsAllowed("machine/oee"))
	assert.False(t, reg.IsAllowed(""))
}

func TestTopicRegistry_Topics(t *testing.T) {
	input := []string{"a/b", "c/d", "e/f"}
	reg, err := NewTopicRegistry(input)
	require.NoError(t, err)

	got := reg.Topics()
	assert.ElementsMatch(t, input, got)
}

func TestNewTopicRegistry_FiveTopics(t *testing.T) {
	topics := []string{"a", "b", "c", "d", "e"}
	reg, err := NewTopicRegistry(topics)
	require.NoError(t, err)
	assert.Equal(t, 5, len(reg.Topics()))
}

// --- Ownership tests ---

func TestClaim_FirstClientBecomesOwner(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status", "machine/alarm"})
	require.NoError(t, err)

	err = reg.Claim("machine/status", "client-A")
	assert.NoError(t, err)
	assert.Equal(t, "client-A", reg.Owner("machine/status"))
}

func TestClaim_SameClientIdempotent(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status"})
	require.NoError(t, err)

	require.NoError(t, reg.Claim("machine/status", "client-A"))
	err = reg.Claim("machine/status", "client-A")
	assert.NoError(t, err)
	assert.Equal(t, "client-A", reg.Owner("machine/status"))
}

func TestClaim_DifferentClientRejected(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status"})
	require.NoError(t, err)

	require.NoError(t, reg.Claim("machine/status", "client-A"))
	err = reg.Claim("machine/status", "client-B")
	assert.ErrorIs(t, err, ErrTopicOwnedByAnother)
	assert.Equal(t, "client-A", reg.Owner("machine/status"))
}

func TestClaim_DisallowedTopic(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status"})
	require.NoError(t, err)

	err = reg.Claim("not/allowed", "client-A")
	assert.ErrorIs(t, err, ErrTopicNotAllowed)
}

func TestClaim_ClientOwnsMultipleTopics(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status", "machine/alarm", "machine/oee"})
	require.NoError(t, err)

	require.NoError(t, reg.Claim("machine/status", "client-A"))
	require.NoError(t, reg.Claim("machine/alarm", "client-A"))
	require.NoError(t, reg.Claim("machine/oee", "client-A"))

	assert.Equal(t, "client-A", reg.Owner("machine/status"))
	assert.Equal(t, "client-A", reg.Owner("machine/alarm"))
	assert.Equal(t, "client-A", reg.Owner("machine/oee"))
}

func TestRelease_FreesAllTopicsForClient(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status", "machine/alarm"})
	require.NoError(t, err)

	require.NoError(t, reg.Claim("machine/status", "client-A"))
	require.NoError(t, reg.Claim("machine/alarm", "client-A"))

	reg.Release("client-A")

	assert.Empty(t, reg.Owner("machine/status"))
	assert.Empty(t, reg.Owner("machine/alarm"))
}

func TestRelease_DoesNotAffectOtherClients(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status", "machine/alarm"})
	require.NoError(t, err)

	require.NoError(t, reg.Claim("machine/status", "client-A"))
	require.NoError(t, reg.Claim("machine/alarm", "client-B"))

	reg.Release("client-A")

	assert.Empty(t, reg.Owner("machine/status"))
	assert.Equal(t, "client-B", reg.Owner("machine/alarm"))
}

func TestRelease_AllowsNewOwnerAfterRelease(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status"})
	require.NoError(t, err)

	require.NoError(t, reg.Claim("machine/status", "client-A"))
	reg.Release("client-A")

	err = reg.Claim("machine/status", "client-B")
	assert.NoError(t, err)
	assert.Equal(t, "client-B", reg.Owner("machine/status"))
}

func TestRelease_NoopForUnknownClient(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status"})
	require.NoError(t, err)

	require.NoError(t, reg.Claim("machine/status", "client-A"))
	reg.Release("unknown-client")

	assert.Equal(t, "client-A", reg.Owner("machine/status"))
}

func TestOwner_UnownedTopicReturnsEmpty(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status"})
	require.NoError(t, err)

	assert.Empty(t, reg.Owner("machine/status"))
}

func TestClaim_ConcurrentAccess(t *testing.T) {
	reg, err := NewTopicRegistry([]string{"machine/status"})
	require.NoError(t, err)

	const goroutines = 50
	results := make(chan error, goroutines)

	for i := range goroutines {
		go func(id int) {
			results <- reg.Claim("machine/status", fmt.Sprintf("client-%d", id))
		}(i)
	}

	var successes, failures int
	for range goroutines {
		if err := <-results; err != nil {
			failures++
		} else {
			successes++
		}
	}

	// Exactly one client should win ownership
	assert.Equal(t, 1, successes, "exactly one goroutine should claim ownership")
	assert.Equal(t, goroutines-1, failures)
	assert.NotEmpty(t, reg.Owner("machine/status"))
}
