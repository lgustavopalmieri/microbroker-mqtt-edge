package queue

import "fmt"

func Push(msg InboundMessage) {
	select {
	case buffer <- msg:
	default:
		fmt.Println("⚠️ Buffer full — message dropped")
	}
}
