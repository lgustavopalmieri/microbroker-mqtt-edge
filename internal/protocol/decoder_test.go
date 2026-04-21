package protocol

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildConnectData builds raw CONNECT variable header + payload bytes.
func buildConnectData(clientID, username, password string, keepAlive uint16, withAuth bool) []byte {
	var buf []byte

	// Protocol Name: "MQTT"
	buf = append(buf, 0x00, 0x04, 'M', 'Q', 'T', 'T')
	// Protocol Level: 4
	buf = append(buf, 0x04)

	// Connect Flags
	flags := byte(0x02) // Clean Session
	if withAuth {
		flags |= 0xC0 // Username + Password flags
	}
	buf = append(buf, flags)

	// Keep Alive
	buf = append(buf, byte(keepAlive>>8), byte(keepAlive&0xFF))

	// Client ID
	buf = append(buf, WriteUTF8String(clientID)...)

	// Username + Password
	if withAuth {
		buf = append(buf, WriteUTF8String(username)...)
		// Password as binary data (2-byte length + content)
		passBytes := []byte(password)
		buf = append(buf, byte(len(passBytes)>>8), byte(len(passBytes)&0xFF))
		buf = append(buf, passBytes...)
	}

	return buf
}

func TestDecodeConnect(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *ConnectPacket
		wantErr error
	}{
		{
			name: "valid connect with auth",
			data: buildConnectData("client1", "user", "pass", 60, true),
			want: &ConnectPacket{
				ProtocolName:  "MQTT",
				ProtocolLevel: 4,
				ConnectFlags:  0xC2, // Username + Password + CleanSession
				KeepAlive:     60,
				ClientID:      "client1",
				Username:      "user",
				Password:      []byte("pass"),
			},
		},
		{
			name: "valid connect without auth",
			data: buildConnectData("device01", "", "", 30, false),
			want: &ConnectPacket{
				ProtocolName:  "MQTT",
				ProtocolLevel: 4,
				ConnectFlags:  0x02, // CleanSession only
				KeepAlive:     30,
				ClientID:      "device01",
			},
		},
		{
			name: "connect with will",
			data: func() []byte {
				var buf []byte
				buf = append(buf, 0x00, 0x04, 'M', 'Q', 'T', 'T') // Protocol Name
				buf = append(buf, 0x04)                           // Protocol Level
				buf = append(buf, 0x0E)                           // Will Flag + Will QoS 1 + CleanSession
				buf = append(buf, 0x00, 0x3C)                     // Keep Alive = 60
				buf = append(buf, WriteUTF8String("client1")...)
				buf = append(buf, WriteUTF8String("will/topic")...)
				willMsg := []byte("goodbye")
				buf = append(buf, byte(len(willMsg)>>8), byte(len(willMsg)&0xFF))
				buf = append(buf, willMsg...)
				return buf
			}(),
			want: &ConnectPacket{
				ProtocolName:  "MQTT",
				ProtocolLevel: 4,
				ConnectFlags:  0x0E,
				KeepAlive:     60,
				ClientID:      "client1",
				WillTopic:     "will/topic",
				WillMessage:   []byte("goodbye"),
			},
		},
		{
			name: "invalid protocol name",
			data: func() []byte {
				var buf []byte
				buf = append(buf, 0x00, 0x04, 'M', 'Q', 'X', 'X')
				buf = append(buf, 0x04, 0x02, 0x00, 0x3C)
				buf = append(buf, WriteUTF8String("c1")...)
				return buf
			}(),
			wantErr: ErrInvalidProtocol,
		},
		{
			name: "invalid protocol level",
			data: func() []byte {
				var buf []byte
				buf = append(buf, 0x00, 0x04, 'M', 'Q', 'T', 'T')
				buf = append(buf, 0x05) // level 5 instead of 4
				buf = append(buf, 0x02, 0x00, 0x3C)
				buf = append(buf, WriteUTF8String("c1")...)
				return buf
			}(),
			wantErr: ErrInvalidProtocol,
		},
		{
			name: "invalid reserved bit",
			data: func() []byte {
				var buf []byte
				buf = append(buf, 0x00, 0x04, 'M', 'Q', 'T', 'T')
				buf = append(buf, 0x04)
				buf = append(buf, 0x03) // reserved bit = 1
				buf = append(buf, 0x00, 0x3C)
				buf = append(buf, WriteUTF8String("c1")...)
				return buf
			}(),
			wantErr: ErrInvalidReservedBit,
		},
		{
			name:    "truncated data - empty",
			data:    []byte{},
			wantErr: ErrTruncatedData,
		},
		{
			name:    "truncated data - partial protocol name",
			data:    []byte{0x00, 0x04, 'M', 'Q'},
			wantErr: ErrTruncatedData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeConnect(tt.data)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.ProtocolName, got.ProtocolName)
			assert.Equal(t, tt.want.ProtocolLevel, got.ProtocolLevel)
			assert.Equal(t, tt.want.ConnectFlags, got.ConnectFlags)
			assert.Equal(t, tt.want.KeepAlive, got.KeepAlive)
			assert.Equal(t, tt.want.ClientID, got.ClientID)
			assert.Equal(t, tt.want.Username, got.Username)
			assert.Equal(t, tt.want.Password, got.Password)
			assert.Equal(t, tt.want.WillTopic, got.WillTopic)
			assert.Equal(t, tt.want.WillMessage, got.WillMessage)
		})
	}
}

func TestDecodePublish(t *testing.T) {
	tests := []struct {
		name    string
		header  FixedHeader
		data    []byte
		want    *PublishPacket
		wantErr error
	}{
		{
			name:   "qos 0 - no packet id",
			header: FixedHeader{PacketType: PUBLISH, Flags: 0x00}, // QoS 0, no DUP, no Retain
			data: func() []byte {
				var buf []byte
				buf = append(buf, WriteUTF8String("machine/status")...)
				buf = append(buf, []byte(`{"temp":42}`)...)
				return buf
			}(),
			want: &PublishPacket{
				QoS:       0,
				TopicName: "machine/status",
				Payload:   []byte(`{"temp":42}`),
			},
		},
		{
			name:   "qos 1 - with packet id",
			header: FixedHeader{PacketType: PUBLISH, Flags: 0x02}, // QoS 1
			data: func() []byte {
				var buf []byte
				buf = append(buf, WriteUTF8String("machine/alarm")...)
				buf = append(buf, 0x00, 0x0A) // PacketID = 10
				buf = append(buf, []byte(`{"alarm":true}`)...)
				return buf
			}(),
			want: &PublishPacket{
				QoS:       1,
				TopicName: "machine/alarm",
				PacketID:  10,
				Payload:   []byte(`{"alarm":true}`),
			},
		},
		{
			name:   "qos 1 with dup and retain",
			header: FixedHeader{PacketType: PUBLISH, Flags: 0x0B}, // DUP=1, QoS=1, Retain=1
			data: func() []byte {
				var buf []byte
				buf = append(buf, WriteUTF8String("t")...)
				buf = append(buf, 0x00, 0x01) // PacketID = 1
				buf = append(buf, 0xFF)
				return buf
			}(),
			want: &PublishPacket{
				DUP:       true,
				QoS:       1,
				Retain:    true,
				TopicName: "t",
				PacketID:  1,
				Payload:   []byte{0xFF},
			},
		},
		{
			name:    "invalid qos 3",
			header:  FixedHeader{PacketType: PUBLISH, Flags: 0x06}, // QoS 3
			data:    WriteUTF8String("topic"),
			wantErr: ErrInvalidQoS,
		},
		{
			name:   "empty topic",
			header: FixedHeader{PacketType: PUBLISH, Flags: 0x00},
			data: func() []byte {
				return WriteUTF8String("")
			}(),
			wantErr: ErrEmptyTopicName,
		},
		{
			name:   "empty payload is valid",
			header: FixedHeader{PacketType: PUBLISH, Flags: 0x00},
			data:   WriteUTF8String("machine/status"),
			want: &PublishPacket{
				QoS:       0,
				TopicName: "machine/status",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodePublish(tt.header, tt.data)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.DUP, got.DUP)
			assert.Equal(t, tt.want.QoS, got.QoS)
			assert.Equal(t, tt.want.Retain, got.Retain)
			assert.Equal(t, tt.want.TopicName, got.TopicName)
			assert.Equal(t, tt.want.PacketID, got.PacketID)
			assert.Equal(t, tt.want.Payload, got.Payload)
		})
	}
}

func TestDecodeSubscribe(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *SubscribePacket
		wantErr error
	}{
		{
			name: "single subscription",
			data: func() []byte {
				var buf []byte
				buf = append(buf, 0x00, 0x0A) // PacketID = 10
				buf = append(buf, WriteUTF8String("machine/status")...)
				buf = append(buf, 0x01) // QoS 1
				return buf
			}(),
			want: &SubscribePacket{
				PacketID: 10,
				Subscriptions: []Subscription{
					{TopicFilter: "machine/status", QoS: 1},
				},
			},
		},
		{
			name: "multiple subscriptions",
			data: func() []byte {
				var buf []byte
				buf = append(buf, 0x00, 0x05) // PacketID = 5
				buf = append(buf, WriteUTF8String("a/b")...)
				buf = append(buf, 0x00) // QoS 0
				buf = append(buf, WriteUTF8String("c/d")...)
				buf = append(buf, 0x02) // QoS 2
				return buf
			}(),
			want: &SubscribePacket{
				PacketID: 5,
				Subscriptions: []Subscription{
					{TopicFilter: "a/b", QoS: 0},
					{TopicFilter: "c/d", QoS: 2},
				},
			},
		},
		{
			name:    "empty payload",
			data:    []byte{0x00, 0x01}, // just packet ID, no subscriptions
			wantErr: ErrEmptyPayload,
		},
		{
			name: "invalid qos",
			data: func() []byte {
				var buf []byte
				buf = append(buf, 0x00, 0x01)
				buf = append(buf, WriteUTF8String("topic")...)
				buf = append(buf, 0x03) // QoS 3 invalid
				return buf
			}(),
			wantErr: ErrInvalidQoS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeSubscribe(tt.data)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.PacketID, got.PacketID)
			assert.Equal(t, tt.want.Subscriptions, got.Subscriptions)
		})
	}
}
