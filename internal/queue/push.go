package queue

import "fmt"

func Push(msg InboundMessage) {
	select {
	case buffer <- msg:
		// ok
	default:
		fmt.Println("⚠️ Buffer full — message dropped")
	}
}
