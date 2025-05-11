package broker //  escuta conexões TCP

import (
	"bufio"
	"fmt"
	"log"
	"microbroker-mqtt-edge/internal/mqtt"
	"net"
)

func ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("⚠️ Accept error: %v", err)
			continue
		}

		log.Printf("📡 New connection from %s", conn.RemoteAddr())

		go handleClient(conn)
	}
}

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
	conn.Write([]byte{0x20, 0x02, 0x00, 0x00}) // CONNACK

	for {
		header, remLen, err := mqtt.ReadPacket(reader)
		if err != nil {
			log.Printf("🔌 Disconnected: %v", err)
			return
		}

		packetType := header >> 4

		switch packetType {
		case mqtt.PacketPingreq:
			conn.Write([]byte{0xD0, 0x00})
			log.Println("📶 PINGREQ received → responded")

		case mqtt.PacketPublish:
			log.Printf("🧾 PUBLISH packet received — remaining length: %d", remLen)
			topic, payload, err := mqtt.ReadPublish(reader, remLen, header)
			if err != nil {
				log.Printf("❌ Failed to parse PUBLISH: %v", err)
				continue
			}
			log.Printf("📤 [%s]: %s", topic, payload)

		default:
			log.Printf("⚠️ Unhandled packet type: %d", packetType)
		}
	}
}
