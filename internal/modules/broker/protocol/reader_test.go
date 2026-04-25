package protocol

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPacket(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		wantHeader FixedHeader
		wantData   []byte
		wantErr    error
	}{
		{
			name: "CONNECT packet",
			input: func() []byte {
				data := buildConnectData("c1", "", "", 60, false)
				var pkt []byte
				pkt = append(pkt, 0x10) // CONNECT type=1, flags=0
				pkt = append(pkt, EncodeRemainingLength(len(data))...)
				pkt = append(pkt, data...)
				return pkt
			}(),
			wantHeader: FixedHeader{PacketType: CONNECT, Flags: 0x00, RemainingLength: len(buildConnectData("c1", "", "", 60, false))},
			wantData:   buildConnectData("c1", "", "", 60, false),
		},
		{
			name: "PUBLISH QoS 0",
			input: func() []byte {
				data := WriteUTF8String("t/1")
				data = append(data, []byte("hello")...)
				var pkt []byte
				pkt = append(pkt, 0x30) // PUBLISH type=3, flags=0
				pkt = append(pkt, EncodeRemainingLength(len(data))...)
				pkt = append(pkt, data...)
				return pkt
			}(),
			wantHeader: FixedHeader{PacketType: PUBLISH, Flags: 0x00},
			wantData: func() []byte {
				data := WriteUTF8String("t/1")
				data = append(data, []byte("hello")...)
				return data
			}(),
		},
		{
			name:       "PINGREQ - zero remaining",
			input:      []byte{0xC0, 0x00},
			wantHeader: FixedHeader{PacketType: PINGREQ, Flags: 0x00, RemainingLength: 0},
			wantData:   nil,
		},
		{
			name:       "DISCONNECT",
			input:      []byte{0xE0, 0x00},
			wantHeader: FixedHeader{PacketType: DISCONNECT, Flags: 0x00, RemainingLength: 0},
			wantData:   nil,
		},
		{
			name:    "reserved packet type 0",
			input:   []byte{0x00, 0x00},
			wantErr: ErrInvalidPacketType,
		},
		{
			name:    "reserved packet type 15",
			input:   []byte{0xF0, 0x00},
			wantErr: ErrInvalidPacketType,
		},
		{
			name:    "EOF on first byte",
			input:   []byte{},
			wantErr: io.EOF,
		},
		{
			name:    "EOF on body",
			input:   []byte{0x30, 0x05, 0x01, 0x02}, // says 5 bytes remaining, only 2 available
			wantErr: io.ErrUnexpectedEOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.input)
			header, data, err := ReadPacket(reader)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantHeader.PacketType, header.PacketType)
			assert.Equal(t, tt.wantHeader.Flags, header.Flags)
			if tt.wantData != nil {
				assert.Equal(t, tt.wantData, data)
			}
		})
	}
}
