package queue

func startWorkers(count int) {
	for i := 0; i < count; i++ {
		go func(id int) {
			for msg := range buffer {
				processing <- msg
			}
		}(i)
	}
}
