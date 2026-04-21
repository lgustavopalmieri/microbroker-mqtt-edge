package protocol

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeRemainingLength(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    int
		wantErr error
	}{
		// 1-byte values (0-127)
		{"zero", []byte{0x00}, 0, nil},
		{"64", []byte{0x40}, 64, nil},
		{"127", []byte{0x7F}, 127, nil},

		// 2-byte values (128-16383)
		{"128", []byte{0x80, 0x01}, 128, nil},
		{"321", []byte{0xC1, 0x02}, 321, nil},
		{"16383", []byte{0xFF, 0x7F}, 16383, nil},

		// 3-byte values (16384-2097151)
		{"16384", []byte{0x80, 0x80, 0x01}, 16384, nil},
		{"2097151", []byte{0xFF, 0xFF, 0x7F}, 2097151, nil},

		// 4-byte values (2097152-268435455)
		{"2097152", []byte{0x80, 0x80, 0x80, 0x01}, 2097152, nil},
		{"268435455", []byte{0xFF, 0xFF, 0xFF, 0x7F}, 268435455, nil},

		// Error: more than 4 continuation bytes
		{"malformed - 5 continuation bytes", []byte{0x80, 0x80, 0x80, 0x80, 0x01}, 0, ErrMalformedRemainingLength},

		// Error: EOF
		{"eof on first byte", []byte{}, 0, io.EOF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.input)
			got, err := DecodeRemainingLength(reader)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEncodeRemainingLength(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  []byte
	}{
		{"zero", 0, []byte{0x00}},
		{"64", 64, []byte{0x40}},
		{"127", 127, []byte{0x7F}},
		{"128", 128, []byte{0x80, 0x01}},
		{"321", 321, []byte{0xC1, 0x02}},
		{"16383", 16383, []byte{0xFF, 0x7F}},
		{"16384", 16384, []byte{0x80, 0x80, 0x01}},
		{"2097151", 2097151, []byte{0xFF, 0xFF, 0x7F}},
		{"2097152", 2097152, []byte{0x80, 0x80, 0x80, 0x01}},
		{"268435455", 268435455, []byte{0xFF, 0xFF, 0xFF, 0x7F}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeRemainingLength(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRemainingLengthRoundtrip(t *testing.T) {
	values := []int{0, 1, 64, 127, 128, 321, 16383, 16384, 2097151, 2097152, 268435455}
	for _, v := range values {
		encoded := EncodeRemainingLength(v)
		decoded, err := DecodeRemainingLength(bytes.NewReader(encoded))
		require.NoError(t, err, "roundtrip failed for %d", v)
		assert.Equal(t, v, decoded, "roundtrip mismatch for %d", v)
	}
}

// --- UTF-8 String Codec Tests (T3) ---

func TestReadUTF8String(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{"empty string", []byte{0x00, 0x00}, "", false},
		{"short string", append([]byte{0x00, 0x05}, []byte("hello")...), "hello", false},
		{"utf8 multibyte", append([]byte{0x00, 0x05}, []byte("café")...), "café", false},
		{"longer string", append([]byte{0x00, 0x0C}, []byte("machine/data")...), "machine/data", false},
		{"truncated length", []byte{0x00}, "", true},
		{"truncated content", []byte{0x00, 0x05, 0x68, 0x65}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.input)
			got, err := ReadUTF8String(reader)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWriteUTF8String(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []byte
	}{
		{"empty", "", []byte{0x00, 0x00}},
		{"hello", "hello", append([]byte{0x00, 0x05}, []byte("hello")...)},
		{"topic path", "machine/status", append([]byte{0x00, 0x0E}, []byte("machine/status")...)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WriteUTF8String(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUTF8StringRoundtrip(t *testing.T) {
	strings := []string{"", "hello", "machine/status", "café", "日本語"}
	for _, s := range strings {
		encoded := WriteUTF8String(s)
		decoded, err := ReadUTF8String(bytes.NewReader(encoded))
		require.NoError(t, err, "roundtrip failed for %q", s)
		assert.Equal(t, s, decoded, "roundtrip mismatch for %q", s)
	}
}

func TestReadBinaryData(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    []byte
		wantErr bool
	}{
		{"empty data", []byte{0x00, 0x00}, []byte{}, false},
		{"some bytes", []byte{0x00, 0x03, 0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}, false},
		{"truncated length", []byte{0x00}, nil, true},
		{"truncated content", []byte{0x00, 0x05, 0x01}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.input)
			got, err := ReadBinaryData(reader)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
