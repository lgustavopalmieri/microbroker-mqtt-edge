package main

import (
	"log"
	"microbroker-mqtt-edge/internal/broker"
	"microbroker-mqtt-edge/internal/queue"
)

func main() {
	log.Println("🚀 Starting micromqttd on :6081...")

	queue.InitQueue(100000)
	queue.StartWorker(2)

	err := broker.ListenAndServe(":6081")
	if err != nil {
		log.Fatalf("❌ Broker error: %v", err)
	}
}
