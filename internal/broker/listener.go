package broker // listen TCP connections

import (
	"bufio"
	"log"
	"microbroker-mqtt-edge/internal/mqtt"
	"microbroker-mqtt-edge/internal/queue"
	"net"
)

func handleClient(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	header, _, err := mqtt.ReadPacket(reader)
	packetType := header >> 4

	if err != nil {
		log.Printf("❌ Failed to read packet: %v", err)
		return
	}

	if packetType != mqtt.PacketConnect {
		log.Printf("❌ Invalid first packet: %d", packetType)
		return
	}

	log.Println("✅ CONNECT received")
	if err := mqtt.SendConnack(conn); err != nil {
		log.Printf("❌ Failed to send CONNACK: %v", err)
		return
	}

	for {
		header, remLen, err := mqtt.ReadPacket(reader)
		if err != nil {
			log.Printf("🔌 Disconnected: %v", err)
			return
		}

		packetType := header >> 4

		switch packetType {
		case mqtt.PacketPingreq:
			if err := mqtt.SendPingresp(conn); err != nil {
				log.Printf("❌ Failed to send PINGRESP: %v", err)
			} else {
				log.Println("📶 PINGREQ received → responded")
			}

		case mqtt.PacketPublish:
			topic, payload, err := mqtt.ReadPublish(reader, remLen, header)
			if err != nil {
				log.Printf("❌ Failed to parse PUBLISH: %v", err)
				continue
			}

			queue.Push(queue.InboundMessage{
				Topic:   topic,
				Payload: []byte(payload),
			})

			log.Printf("📤 [%s]: %s", topic, payload)

		default:
			log.Printf("⚠️ Unhandled packet type: %d", packetType)
		}
	}
}
