package protocol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeConnack(t *testing.T) {
	tests := []struct {
		name           string
		sessionPresent bool
		returnCode     ConnackReturnCode
		want           []byte
	}{
		{"accepted", false, ConnAccepted, []byte{0x20, 0x02, 0x00, 0x00}},
		{"accepted with session present", true, ConnAccepted, []byte{0x20, 0x02, 0x01, 0x00}},
		{"bad auth", false, ConnRefusedBadAuth, []byte{0x20, 0x02, 0x00, 0x04}},
		{"not authorized", false, ConnRefusedNotAuth, []byte{0x20, 0x02, 0x00, 0x05}},
		{"unacceptable protocol", false, ConnRefusedProtocol, []byte{0x20, 0x02, 0x00, 0x01}},
		{"identifier rejected", false, ConnRefusedIdentifier, []byte{0x20, 0x02, 0x00, 0x02}},
		{"server unavailable", false, ConnRefusedUnavailable, []byte{0x20, 0x02, 0x00, 0x03}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeConnack(tt.sessionPresent, tt.returnCode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEncodePuback(t *testing.T) {
	tests := []struct {
		name     string
		packetID uint16
		want     []byte
	}{
		{"packet id 1", 1, []byte{0x40, 0x02, 0x00, 0x01}},
		{"packet id 10", 10, []byte{0x40, 0x02, 0x00, 0x0A}},
		{"packet id 256", 256, []byte{0x40, 0x02, 0x01, 0x00}},
		{"packet id max", 65535, []byte{0x40, 0x02, 0xFF, 0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodePuback(tt.packetID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEncodeSuback(t *testing.T) {
	tests := []struct {
		name        string
		packetID    uint16
		returnCodes []byte
		want        []byte
	}{
		{
			"single qos 0",
			10,
			[]byte{0x00},
			[]byte{0x90, 0x03, 0x00, 0x0A, 0x00},
		},
		{
			"single qos 1",
			1,
			[]byte{0x01},
			[]byte{0x90, 0x03, 0x00, 0x01, 0x01},
		},
		{
			"two return codes",
			5,
			[]byte{0x00, 0x02},
			[]byte{0x90, 0x04, 0x00, 0x05, 0x00, 0x02},
		},
		{
			"failure code",
			1,
			[]byte{0x80},
			[]byte{0x90, 0x03, 0x00, 0x01, 0x80},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeSuback(tt.packetID, tt.returnCodes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEncodePingresp(t *testing.T) {
	got := EncodePingresp()
	assert.Equal(t, []byte{0xD0, 0x00}, got)
}
