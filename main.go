package main

import (
	"log"
	"microbroker-mqtt-edge/internal/broker"
	"microbroker-mqtt-edge/internal/queue"
)

func main() {
	log.Println("🚀 Starting micromqttd on :6081...")

	bufferSize := 100000
	workers := 2
	
	queue.InitQueue(bufferSize)
	queue.StartWorker(workers)

	err := broker.ListenAndServe(":6081")
	if err != nil {
		log.Fatalf("❌ Broker error: %v", err)
	}
}

// documentar muito top e vamos fazer TESTES!!!
// fizemos os manuais agora mas vamos fazer escritos,
// e também vamos fazer testes de disparar mensagens
// concorrentes de 10 tópicos diferentes em 300 por segundo cada 1.
