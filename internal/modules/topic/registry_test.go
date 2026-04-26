package topic

import (
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
