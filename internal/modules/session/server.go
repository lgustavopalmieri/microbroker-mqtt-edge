package session

import (
	"context"
	"net"
	"time"

	"microbroker-mqtt-edge/internal/modules/session/domain"
)

// Message represents a data point received from a client via PUBLISH.
// This is the output type that flows from session → ingestion.
type Message struct {
	ClientID  string
	Topic     string
	Payload   []byte
	Timezone  string
	Timestamp time.Time
}

// Server is the TCP listener that accepts MQTT client connections.
type Server struct {
	address  string
	listener net.Listener
	connMgr  *ConnectionManager
	auth     Authenticator
	topics   *domain.TopicRegistry
	msgChan  chan<- Message
	timezone string
	logger   Logger
	ready    chan struct{} // closed when listener is ready
}

// NewServer creates a new MQTT TCP server with all dependencies injected.
func NewServer(
	address string,
	connMgr *ConnectionManager,
	auth Authenticator,
	topics *domain.TopicRegistry,
	msgChan chan<- Message,
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

// ListenAndServe starts the TCP listener and accept loop.
// Blocks until the context is cancelled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	var err error
	s.listener, err = net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	s.logger.Info("broker started", "address", s.address)

	close(s.ready) // signal that listener is ready

	go func() {
		<-ctx.Done()
		s.listener.Close()
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				s.logger.Error("accept error", "error", err)
				continue
			}
		}

		if !s.connMgr.CanAccept() {
			conn.Close()
			s.logger.Warn("connection rejected: max clients reached")
			continue
		}

		go s.handleConnection(ctx, conn)
	}
}

// Close stops the listener.
func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// Addr returns the listener address (useful for tests with port 0).
func (s *Server) Addr() net.Addr {
	if s.listener != nil {
		return s.listener.Addr()
	}
	return nil
}

// Ready returns a channel that is closed when the server's listener is ready.
func (s *Server) Ready() <-chan struct{} {
	return s.ready
}
