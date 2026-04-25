package server

import (
	"context"
	"time"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/modules/connection/client"
	"microbroker-mqtt-edge/internal/modules/protocol"
)

func (s *Server) handlePublish(ctx context.Context, cl *client.Client, header protocol.FixedHeader, data []byte) {
	pkt, err := protocol.DecodePublish(header, data)
	if err != nil {
		s.logger.Warn("invalid PUBLISH", "client", cl.ID, "error", err)
		return
	}

	if !s.topics.IsAllowed(pkt.TopicName) {
		s.logger.Debug("publish to disallowed topic", "client", cl.ID, "topic", pkt.TopicName)
		// Still send PUBACK if QoS 1 to avoid client retries
		if pkt.QoS == 1 {
			cl.Write(protocol.EncodePuback(pkt.PacketID))
		}
		return
	}

	msg := message.Message{
		ClientID:  cl.ID,
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
		cl.Write(protocol.EncodePuback(pkt.PacketID))
	}
}
