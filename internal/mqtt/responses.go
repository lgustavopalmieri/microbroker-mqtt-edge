package mqtt

import (
	"net"
)

// SendConnack sends a fixed CONNACK response
func SendConnack(conn net.Conn) error {
	// CONNACK: type 0x20, remaining length 0x02, flags 0x00, return code 0x00
	_, err := conn.Write([]byte{0x20, 0x02, 0x00, 0x00})
	return err
}

// SendPingresp sends a fixed PINGRESP response
func SendPingresp(conn net.Conn) error {
	// PINGRESP: type 0xD0, remaining length 0x00
	_, err := conn.Write([]byte{0xD0, 0x00})
	return err
}
