package queue

import "fmt"

func startProcessors(processors int) {
	for i := 0; i < processors; i++ {
		go func(id int) {
			for msg := range processing {
				fmt.Printf("⚙️ Handler %d → %s\n", id, msg.Topic)

				mu.Lock()
				count++
				mu.Unlock()

			}
		}(i)
	}
}
