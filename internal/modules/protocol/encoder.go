package protocol

// EncodeConnack builds a CONNACK packet.
func EncodeConnack(sessionPresent bool, returnCode ConnackReturnCode) []byte {
	sp := byte(0x00)
	if sessionPresent {
		sp = 0x01
	}
	return []byte{0x20, 0x02, sp, byte(returnCode)}
}

// EncodePuback builds a PUBACK packet for the given packet identifier.
func EncodePuback(packetID uint16) []byte {
	return []byte{
		0x40, 0x02,
		byte(packetID >> 8),
		byte(packetID & 0xFF),
	}
}

// EncodeSuback builds a SUBACK packet with the given packet identifier and return codes.
// Each return code is 0x00 (QoS 0), 0x01 (QoS 1), 0x02 (QoS 2), or 0x80 (failure).
func EncodeSuback(packetID uint16, returnCodes []byte) []byte {
	remainingLength := 2 + len(returnCodes) // 2 for packet ID + return codes
	buf := []byte{0x90}
	buf = append(buf, EncodeRemainingLength(remainingLength)...)
	buf = append(buf, byte(packetID>>8), byte(packetID&0xFF))
	buf = append(buf, returnCodes...)
	return buf
}

// EncodePingresp builds a PINGRESP packet.
func EncodePingresp() []byte {
	return []byte{0xD0, 0x00}
}
