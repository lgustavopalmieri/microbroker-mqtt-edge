package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all broker configuration loaded from environment variables.
type Config struct {
	Host            string
	Port            int
	HTTPPort        int
	Username        string
	Password        string
	Topics          []string
	MaxClients      int
	QueueBufferSize int
	DBPath          string
	Timezone        string
}

// Defaults
const (
	defaultHost            = "0.0.0.0"
	defaultPort            = 1883
	defaultHTTPPort        = 8080
	defaultMaxClients      = 5
	defaultQueueBufferSize = 10000
	defaultDBPath          = "./data/broker.db"
	defaultTimezone        = "UTC"
)

// Validation errors
var (
	ErrTopicsEmpty     = errors.New("config: BROKER_TOPICS must not be empty")
	ErrMaxClientsRange = errors.New("config: BROKER_MAX_CLIENTS must be between 1 and 5")
	ErrUsernameEmpty   = errors.New("config: BROKER_USERNAME must not be empty")
	ErrPasswordEmpty   = errors.New("config: BROKER_PASSWORD must not be empty")
)

// Address returns the "host:port" string for the TCP listener.
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// HTTPAddress returns the "host:port" string for the HTTP API listener.
func (c *Config) HTTPAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.HTTPPort)
}

// Load reads configuration from environment variables, applies defaults,
// trims whitespace, and validates all required fields.
func Load() (*Config, error) {
	cfg := &Config{
		Host:            envOrDefault("BROKER_HOST", defaultHost),
		Port:            envIntOrDefault("BROKER_PORT", defaultPort),
		HTTPPort:        envIntOrDefault("BROKER_HTTP_PORT", defaultHTTPPort),
		Username:        strings.TrimSpace(os.Getenv("BROKER_USERNAME")),
		Password:        strings.TrimSpace(os.Getenv("BROKER_PASSWORD")),
		MaxClients:      envIntOrDefault("BROKER_MAX_CLIENTS", defaultMaxClients),
		QueueBufferSize: envIntOrDefault("BROKER_QUEUE_BUFFER_SIZE", defaultQueueBufferSize),
		DBPath:          envOrDefault("BROKER_DB_PATH", defaultDBPath),
		Timezone:        envOrDefault("BROKER_TIMEZONE", defaultTimezone),
	}

	// Parse topics: split by comma, trim each, ignore empty
	cfg.Topics = parseTopics(os.Getenv("BROKER_TOPICS"))

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if len(c.Topics) == 0 {
		return ErrTopicsEmpty
	}
	if c.MaxClients < 1 || c.MaxClients > 5 {
		return ErrMaxClientsRange
	}
	if c.Username == "" {
		return ErrUsernameEmpty
	}
	if c.Password == "" {
		return ErrPasswordEmpty
	}
	return nil
}

func parseTopics(raw string) []string {
	parts := strings.Split(raw, ",")
	var topics []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			topics = append(topics, t)
		}
	}
	return topics
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func envIntOrDefault(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
