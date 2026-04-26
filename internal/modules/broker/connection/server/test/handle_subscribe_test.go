package test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/testutil"
)

func TestHandleSubscribe_AllowedAndDisallowed(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "subscriber")
	defer conn.Close()

	conn.Write(testutil.BuildSubscribePacket(10, []string{"machine/status", "not/allowed"}, []byte{0x01, 0x00}))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	headerBuf := make([]byte, 2)
	_, err := io.ReadFull(conn, headerBuf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x90), headerBuf[0])

	remainingLen := int(headerBuf[1])
	body := make([]byte, remainingLen)
	_, err = io.ReadFull(conn, body)
	require.NoError(t, err)

	reader := bytes.NewReader(body)
	pidBuf := make([]byte, 2)
	_, err = io.ReadFull(reader, pidBuf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x00), pidBuf[0])
	assert.Equal(t, byte(0x0A), pidBuf[1])

	rc1, _ := reader.ReadByte()
	rc2, _ := reader.ReadByte()
	assert.Equal(t, byte(0x01), rc1)
	assert.Equal(t, byte(0x80), rc2)
}

func TestHandleSubscribe_AllAllowed(t *testing.T) {
	ts, _, cancel := setupTestServer(t)
	defer cancel()

	conn := connectClient(t, ts, "sub-all")
	defer conn.Close()

	conn.Write(testutil.BuildSubscribePacket(20, []string{"machine/status", "machine/alarm"}, []byte{0x00, 0x01}))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	headerBuf := make([]byte, 2)
	_, err := io.ReadFull(conn, headerBuf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x90), headerBuf[0])

	remainingLen := int(headerBuf[1])
	body := make([]byte, remainingLen)
	_, err = io.ReadFull(conn, body)
	require.NoError(t, err)

	reader := bytes.NewReader(body)
	pidBuf := make([]byte, 2)
	_, err = io.ReadFull(reader, pidBuf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x00), pidBuf[0])
	assert.Equal(t, byte(0x14), pidBuf[1])

	rc1, _ := reader.ReadByte()
	rc2, _ := reader.ReadByte()
	assert.Equal(t, byte(0x00), rc1)
	assert.Equal(t, byte(0x01), rc2)
}
