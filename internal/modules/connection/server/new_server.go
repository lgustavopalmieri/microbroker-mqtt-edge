package server

import (
	"net"
	"time"

	"microbroker-mqtt-edge/internal/common/message"
	clientmanager "microbroker-mqtt-edge/internal/modules/connection/client_manager"
	topicdomain "microbroker-mqtt-edge/internal/modules/topic/domain"
)

const connectTimeout = 5 * time.Second

// Logger defines the logging interface used by the server.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
	Debug(msg string, args ...any)
}

// Authenticator validates client credentials.
type Authenticator interface {
	Authenticate(username, password string) bool
}

// Server is the TCP listener that accepts MQTT client connections.
type Server struct {
	address  string
	listener net.Listener
	connMgr  *clientmanager.ClientManager
	auth     Authenticator
	topics   *topicdomain.TopicRegistry
	msgChan  chan<- message.Message
	timezone string
	logger   Logger
	ready    chan struct{} // closed when listener is ready
}

// NewServer creates a new MQTT TCP server with all dependencies injected.
func NewServer(
	address string,
	connMgr *clientmanager.ClientManager,
	auth Authenticator,
	topics *topicdomain.TopicRegistry,
	msgChan chan<- message.Message,
	timezone string,
	logger Logger,
) *Server {
	return &Server{
		address:  address,
		connMgr:  connMgr,
		auth:     auth,
		topics:   topics,
		msgChan:  msgChan,
		timezone: timezone,
		logger:   logger,
		ready:    make(chan struct{}),
	}
}
