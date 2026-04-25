package server

import (
	"context"
	"net"

	"microbroker-mqtt-edge/internal/modules/broker/connection/client"
	"microbroker-mqtt-edge/internal/modules/broker/protocol"
)

// readLoop processes packets from a connected client until disconnect or error.
func (s *Server) readLoop(ctx context.Context, cl *client.Client, conn net.Conn) {
	for {
		cl.ResetDeadline()

		header, data, err := protocol.ReadPacket(conn)
		if err != nil {
			select {
			case <-ctx.Done():
			default:
				s.logger.Debug("read error", "client", cl.ID, "error", err)
			}
			return
		}

		switch header.PacketType {
		case protocol.PUBLISH:
			s.handlePublish(ctx, cl, header, data)

		case protocol.SUBSCRIBE:
			s.handleSubscribe(cl, data)

		case protocol.PINGREQ:
			cl.Write(protocol.EncodePingresp())

		case protocol.DISCONNECT:
			return

		default:
			s.logger.Warn("unexpected packet type", "client", cl.ID, "type", header.PacketType.String())
		}
	}
}
