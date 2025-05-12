package queue

import (
	"sync"
)

var (
	buffer     chan InboundMessage
	processing chan InboundMessage
	mu         sync.Mutex
	count      int
)

func StartWorker(workerCount int) {
	startWorkers(workerCount)
	startProcessors(2 * workerCount)
	startLogLoop()
}
