package queue

import (
	"fmt"
	"time"
)

func startLogLoop() {
	go func() {
		for {
			mu.Lock()
			total := count
			mu.Unlock()

			time.Sleep(3 * time.Second)
			fmt.Printf("📈 Total processed: %d\n", total)
		}
	}()
}
