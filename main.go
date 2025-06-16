package main

import (
	"log"
	"microbroker-mqtt-edge/internal/broker"
	"microbroker-mqtt-edge/internal/queue"
)

func main() {
	log.Println("🚀 Starting micromqttd on :6081...")

	bufferSize := 10000
	workers := 2

	queue.InitQueue(bufferSize)
	queue.StartWorker(workers)

	err := broker.ListenAndServe(":6081")
	if err != nil {
		log.Fatalf("❌ Broker error: %v", err)
	}
}
