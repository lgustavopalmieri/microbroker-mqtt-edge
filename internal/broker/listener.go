package broker //  escuta conexões TCP

import (
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

	packetType, _, err := mqtt.ReadPacket(conn)
	if err != nil {
		log.Printf("❌ Failed to read packet: %v", err)
		return
	}

	if packetType != mqtt.PacketConnect {
		log.Printf("❌ Invalid first packet: %d", packetType)
		return
	}

	log.Println("✅ CONNECT received")

	// Envia CONNACK com sucesso
	conn.Write([]byte{0x20, 0x02, 0x00, 0x00})

	// ✅ Loop contínuo para manter conexão
	for {
		packetType, _, err := mqtt.ReadPacket(conn)
		if err != nil {
			log.Printf("🔌 Disconnected: %v", err)
			return
		}

		switch packetType {
		case mqtt.PacketPingreq:
			// Responde com PINGRESP
			conn.Write([]byte{0xD0, 0x00})
			log.Println("📶 PINGREQ received → responded")

		default:
			log.Printf("⚠️ Unhandled packet type: %d\n", packetType)
		}
	}
}
