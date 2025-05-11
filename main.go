package main

import (
	"log"
	"microbroker-mqtt-edge/internal/broker"
)

func main() {
	log.Println("🚀 Starting micromqttd on :6081...")

	err := broker.ListenAndServe(":6081")
	if err != nil {
		log.Fatalf("❌ Broker error: %v", err)
	}
}
