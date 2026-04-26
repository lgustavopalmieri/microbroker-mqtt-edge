package server

import (
	"context"
	"net"
	"time"

	"microbroker-mqtt-edge/internal/modules/broker/connection/client"
	"microbroker-mqtt-edge/internal/modules/broker/protocol"
)

// handleConnection processes the full lifecycle of a single MQTT client connection.
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	// Set initial timeout for CONNECT packet
	conn.SetDeadline(time.Now().Add(connectTimeout))

	// 1. Read first packet — must be CONNECT
	header, data, err := protocol.ReadPacket(conn)
	if err != nil {
		s.logger.Debug("failed to read first packet", "error", err)
		return
	}

	if header.PacketType != protocol.CONNECT {
		s.logger.Warn("first packet is not CONNECT", "type", header.PacketType.String())
		return
	}

	// 2. Decode CONNECT
	connectPkt, err := protocol.DecodeConnect(data)
	if err != nil {
		s.logger.Warn("invalid CONNECT packet", "error", err)
		return
	}

	// 3. Authenticate
	if !connectPkt.HasUsername() || !s.auth.Authenticate(connectPkt.Username, string(connectPkt.Password)) {
		conn.Write(protocol.EncodeConnack(false, protocol.ConnRefusedBadAuth))
		s.logger.Info("auth failed", "client", connectPkt.ClientID)
		return
	}

	// 4. Register client
	cl := client.NewClient(connectPkt.ClientID, conn, connectPkt.KeepAlive)
	if err := s.connMgr.Add(cl); err != nil {
		conn.Write(protocol.EncodeConnack(false, protocol.ConnRefusedUnavailable))
		s.logger.Warn("cannot add client", "client", connectPkt.ClientID, "error", err)
		return
	}
	defer s.connMgr.Remove(cl.ID)
	defer s.topics.Release(cl.ID)

	// 5. Send CONNACK success
	conn.Write(protocol.EncodeConnack(false, protocol.ConnAccepted))
	s.logger.Info("client connected", "client", cl.ID)

	// 6. Read loop
	s.readLoop(ctx, cl, conn)

	s.logger.Info("client disconnected", "client", cl.ID)
}
