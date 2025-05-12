package broker

import (
	"fmt"
	"log"
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
