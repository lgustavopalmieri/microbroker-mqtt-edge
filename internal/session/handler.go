package session

import (
	"context"
	"net"
	"time"

	"microbroker-mqtt-edge/internal/protocol"
	"microbroker-mqtt-edge/internal/session/domain"
)

const connectTimeout = 5 * time.Second

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
	client := domain.NewClient(connectPkt.ClientID, conn, connectPkt.KeepAlive)
	if err := s.connMgr.Add(client); err != nil {
		conn.Write(protocol.EncodeConnack(false, protocol.ConnRefusedUnavailable))
		s.logger.Warn("cannot add client", "client", connectPkt.ClientID, "error", err)
		return
	}
	defer s.connMgr.Remove(client.ID)

	// 5. Send CONNACK success
	conn.Write(protocol.EncodeConnack(false, protocol.ConnAccepted))
	s.logger.Info("client connected", "client", client.ID)

	// 6. Read loop
	s.readLoop(ctx, client, conn)

	s.logger.Info("client disconnected", "client", client.ID)
}

// readLoop processes packets from a connected client until disconnect or error.
func (s *Server) readLoop(ctx context.Context, client *domain.Client, conn net.Conn) {
	for {
		client.ResetDeadline()

		header, data, err := protocol.ReadPacket(conn)
		if err != nil {
			select {
			case <-ctx.Done():
			default:
				s.logger.Debug("read error", "client", client.ID, "error", err)
			}
			return
		}

		switch header.PacketType {
		case protocol.PUBLISH:
			s.handlePublish(ctx, client, header, data)

		case protocol.SUBSCRIBE:
			s.handleSubscribe(client, data)

		case protocol.PINGREQ:
			client.Write(protocol.EncodePingresp())

		case protocol.DISCONNECT:
			return

		default:
			s.logger.Warn("unexpected packet type", "client", client.ID, "type", header.PacketType.String())
		}
	}
}

func (s *Server) handlePublish(ctx context.Context, client *domain.Client, header protocol.FixedHeader, data []byte) {
	pkt, err := protocol.DecodePublish(header, data)
	if err != nil {
		s.logger.Warn("invalid PUBLISH", "client", client.ID, "error", err)
		return
	}

	if !s.topics.IsAllowed(pkt.TopicName) {
		s.logger.Debug("publish to disallowed topic", "client", client.ID, "topic", pkt.TopicName)
		// Still send PUBACK if QoS 1 to avoid client retries
		if pkt.QoS == 1 {
			client.Write(protocol.EncodePuback(pkt.PacketID))
		}
		return
	}

	msg := Message{
		ClientID:  client.ID,
		Topic:     pkt.TopicName,
		Payload:   pkt.Payload,
		Timezone:  s.timezone,
		Timestamp: time.Now(),
	}

	select {
	case s.msgChan <- msg:
	case <-ctx.Done():
		return
	}

	if pkt.QoS == 1 {
		client.Write(protocol.EncodePuback(pkt.PacketID))
	}
}

func (s *Server) handleSubscribe(client *domain.Client, data []byte) {
	pkt, err := protocol.DecodeSubscribe(data)
	if err != nil {
		s.logger.Warn("invalid SUBSCRIBE", "client", client.ID, "error", err)
		return
	}

	returnCodes := make([]byte, len(pkt.Subscriptions))
	for i, sub := range pkt.Subscriptions {
		if s.topics.IsAllowed(sub.TopicFilter) {
			returnCodes[i] = sub.QoS
		} else {
			returnCodes[i] = 0x80 // failure
		}
	}

	client.Write(protocol.EncodeSuback(pkt.PacketID, returnCodes))
}
