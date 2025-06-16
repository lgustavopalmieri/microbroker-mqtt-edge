package queue

import (
	"fmt"
	"time"
)

func InitQueue(size int) {
	buffer = make(chan InboundMessage, size)
	processing = make(chan InboundMessage, size) // secondary channel (fan-out)

	// original buffer monitor
	go func() {
		for {
			time.Sleep(1 * time.Second)
			fmt.Printf("📊 Buffer: %d messages\n", len(buffer))
		}
	}()
}
