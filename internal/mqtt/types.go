package mqtt

const (
	PacketConnect    = 1
	PacketConnack    = 2
	PacketPublish    = 3
	PacketPuback     = 4
	PacketPubrec     = 5
	PacketPubrel     = 6
	PacketPubcomp    = 7
	PacketSubscribe  = 8
	PacketSuback     = 9
	PacketUnsub      = 10
	PacketUnsuback   = 11
	PacketPingreq    = 12
	PacketPingresp   = 13
	PacketDisconnect = 14

	// Responses (control packets)
	ConnackSuccess = "\x20\x02\x00\x00" // CONNACK fixed response
	Pingresp       = "\xD0\x00"         // PINGRESP fixed response
)
