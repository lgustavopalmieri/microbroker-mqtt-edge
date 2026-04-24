package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setEnv is a helper that sets env vars and returns a cleanup function.
func setEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

func validEnv() map[string]string {
	return map[string]string{
		"BROKER_HOST":              "127.0.0.1",
		"BROKER_PORT":              "1884",
		"BROKER_USERNAME":          "admin",
		"BROKER_PASSWORD":          "secret",
		"BROKER_TOPICS":            "machine/status,machine/alarm",
		"BROKER_MAX_CLIENTS":       "3",
		"BROKER_QUEUE_BUFFER_SIZE": "5000",
		"BROKER_DB_PATH":           "/tmp/test.db",
		"BROKER_TIMEZONE":          "America/Sao_Paulo",
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	setEnv(t, validEnv())

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1", cfg.Host)
	assert.Equal(t, 1884, cfg.Port)
	assert.Equal(t, "admin", cfg.Username)
	assert.Equal(t, "secret", cfg.Password)
	assert.Equal(t, []string{"machine/status", "machine/alarm"}, cfg.Topics)
	assert.Equal(t, 3, cfg.MaxClients)
	assert.Equal(t, 5000, cfg.QueueBufferSize)
	assert.Equal(t, "/tmp/test.db", cfg.DBPath)
	assert.Equal(t, "America/Sao_Paulo", cfg.Timezone)
	assert.Equal(t, "127.0.0.1:1884", cfg.Address())
}

func TestLoad_DefaultsApplied(t *testing.T) {
	setEnv(t, map[string]string{
		"BROKER_USERNAME": "admin",
		"BROKER_PASSWORD": "secret",
		"BROKER_TOPICS":   "t/1",
	})

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, defaultHost, cfg.Host)
	assert.Equal(t, defaultPort, cfg.Port)
	assert.Equal(t, defaultMaxClients, cfg.MaxClients)
	assert.Equal(t, defaultQueueBufferSize, cfg.QueueBufferSize)
	assert.Equal(t, defaultDBPath, cfg.DBPath)
	assert.Equal(t, defaultTimezone, cfg.Timezone)
}

func TestLoad_ErrorWhenTopicsEmpty(t *testing.T) {
	setEnv(t, map[string]string{
		"BROKER_USERNAME": "admin",
		"BROKER_PASSWORD": "secret",
		"BROKER_TOPICS":   "",
	})

	_, err := Load()
	assert.ErrorIs(t, err, ErrTopicsEmpty)
}

func TestLoad_ErrorWhenMaxClientsExceedsFive(t *testing.T) {
	setEnv(t, map[string]string{
		"BROKER_USERNAME":    "admin",
		"BROKER_PASSWORD":    "secret",
		"BROKER_TOPICS":      "t/1",
		"BROKER_MAX_CLIENTS": "6",
	})

	_, err := Load()
	assert.ErrorIs(t, err, ErrMaxClientsRange)
}

func TestLoad_ErrorWhenMaxClientsZero(t *testing.T) {
	setEnv(t, map[string]string{
		"BROKER_USERNAME":    "admin",
		"BROKER_PASSWORD":    "secret",
		"BROKER_TOPICS":      "t/1",
		"BROKER_MAX_CLIENTS": "0",
	})

	_, err := Load()
	assert.ErrorIs(t, err, ErrMaxClientsRange)
}

func TestLoad_ErrorWhenUsernameEmpty(t *testing.T) {
	setEnv(t, map[string]string{
		"BROKER_PASSWORD": "secret",
		"BROKER_TOPICS":   "t/1",
	})
	os.Setenv("BROKER_USERNAME", "")

	_, err := Load()
	assert.ErrorIs(t, err, ErrUsernameEmpty)
}

func TestLoad_ErrorWhenPasswordEmpty(t *testing.T) {
	setEnv(t, map[string]string{
		"BROKER_USERNAME": "admin",
		"BROKER_TOPICS":   "t/1",
	})
	os.Setenv("BROKER_PASSWORD", "")

	_, err := Load()
	assert.ErrorIs(t, err, ErrPasswordEmpty)
}

func TestLoad_TrimSpaces(t *testing.T) {
	setEnv(t, map[string]string{
		"BROKER_HOST":     "  127.0.0.1  ",
		"BROKER_USERNAME": "  admin  ",
		"BROKER_PASSWORD": "  secret  ",
		"BROKER_TOPICS":   "  t/1 , t/2  ",
		"BROKER_DB_PATH":  "  /tmp/db  ",
		"BROKER_TIMEZONE": "  UTC  ",
	})

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1", cfg.Host)
	assert.Equal(t, "admin", cfg.Username)
	assert.Equal(t, "secret", cfg.Password)
	assert.Equal(t, []string{"t/1", "t/2"}, cfg.Topics)
	assert.Equal(t, "/tmp/db", cfg.DBPath)
	assert.Equal(t, "UTC", cfg.Timezone)
}

func TestLoad_TopicsWithExtraCommas(t *testing.T) {
	setEnv(t, map[string]string{
		"BROKER_USERNAME": "admin",
		"BROKER_PASSWORD": "secret",
		"BROKER_TOPICS":   "a,,b,,,c,",
	})

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, []string{"a", "b", "c"}, cfg.Topics)
}

func TestLoad_InvalidPortFallsBackToDefault(t *testing.T) {
	setEnv(t, map[string]string{
		"BROKER_USERNAME": "admin",
		"BROKER_PASSWORD": "secret",
		"BROKER_TOPICS":   "t/1",
		"BROKER_PORT":     "not-a-number",
	})

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, defaultPort, cfg.Port)
}
