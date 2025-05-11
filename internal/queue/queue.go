package queue

import (
	"fmt"
	"sync"
	"time"
)

var (
	buffer     chan []byte
	processing chan []byte
	mu         sync.Mutex
	count      int
	processors = 5 // fan-out level
)

func InitQueue(size int) {
	buffer = make(chan []byte, size)
	processing = make(chan []byte, size) // canal secundário (fan-out)

	// Monitor do buffer original
	go func() {
		for {
			time.Sleep(1 * time.Second)
			fmt.Printf("📊 Buffer: %d messages\n", len(buffer))
		}
	}()
}

func Push(msg []byte) {
	select {
	case buffer <- msg:
		// ok
	default:
		fmt.Println("⚠️ Buffer full — message dropped")
	}
}

func StartWorker(workerCount int) {
	// 👷 Workers leves (só transferem)
	for i := 0; i < workerCount; i++ {
		go func(id int) {
			for msg := range buffer {
				processing <- msg
			}
		}(i)
	}

	// 🔥 Processadores pesados (fan-out)
	for i := 0; i < processors; i++ {
		go func(id int) {
			for msg := range processing {
				// Simular processamento pesado
				fmt.Printf("⚙️ Handler %d → %s\n", id, msg)

				mu.Lock()
				count++
				mu.Unlock()
			}
		}(i)
	}

	// Log
	go func() {
		for {
			time.Sleep(3 * time.Second)
			mu.Lock()
			fmt.Printf("📈 Total processed: %d\n", count)
			mu.Unlock()
		}
	}()
}
