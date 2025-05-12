package queue

import (
	"fmt"
	"time"
)

func InitQueue(size int) {
	buffer = make(chan InboundMessage, size)
	processing = make(chan InboundMessage, size) // canal secundário (fan-out)

	// Monitor do buffer original
	go func() {
		for {
			time.Sleep(1 * time.Second)
			fmt.Printf("📊 Buffer: %d messages\n", len(buffer))
		}
	}()
}
