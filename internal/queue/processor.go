package queue

import (
	"fmt"
	"sync"
)

func startProcessors(processors int) {
	for i := 0; i < processors; i++ {
		go func(id int) {
			for msg := range processing {
				fmt.Printf("⚙️ Handler %d → %s\n", id, msg.Topic)
				wg := &sync.WaitGroup{}
				wg.Add(3)

				go saveToDisk(msg, wg)
				go persistToDB(msg, wg)
				go sendToCloud(msg, wg)

				wg.Wait()

				mu.Lock()
				count++
				mu.Unlock()

			}
		}(i)
	}
}

func saveToDisk(msg InboundMessage, wg *sync.WaitGroup) {
	defer wg.Done()
	// Simulate saving to disk
	fmt.Printf("💾 Saving %s to disk\n", msg.Topic)
}

func persistToDB(msg InboundMessage, wg *sync.WaitGroup) {
	defer wg.Done()
	// simulate DB insert
}

func sendToCloud(msg InboundMessage, wg *sync.WaitGroup) {
	defer wg.Done()
	// simulate cloud pub
}
