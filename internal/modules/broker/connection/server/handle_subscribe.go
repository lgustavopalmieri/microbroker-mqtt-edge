package server

import (
	"microbroker-mqtt-edge/internal/modules/broker/connection/client"
	"microbroker-mqtt-edge/internal/modules/broker/protocol"
)

func (s *Server) handleSubscribe(cl *client.Client, data []byte) {
	pkt, err := protocol.DecodeSubscribe(data)
	if err != nil {
		s.logger.Warn("invalid SUBSCRIBE", "client", cl.ID, "error", err)
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

	cl.Write(protocol.EncodeSuback(pkt.PacketID, returnCodes))
}
