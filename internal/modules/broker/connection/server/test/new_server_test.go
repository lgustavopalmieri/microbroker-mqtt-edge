package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"microbroker-mqtt-edge/internal/common/message"
	"microbroker-mqtt-edge/internal/common/observability"
	clientmanager "microbroker-mqtt-edge/internal/modules/broker/connection/client_manager"
	"microbroker-mqtt-edge/internal/modules/broker/connection/server"
	topicdomain "microbroker-mqtt-edge/internal/modules/topic"
)

type stubAuth struct{ allow bool }

func (s stubAuth) Authenticate(_, _ string) bool { return s.allow }

func TestNewServer(t *testing.T) {
	topics, err := topicdomain.NewTopicRegistry([]string{"t/1"})
	require.NoError(t, err)

	msgChan := make(chan message.Message, 1)
	connMgr := clientmanager.NewClientManager(5)

	srv := server.NewServer("127.0.0.1:0", connMgr, stubAuth{true}, topics, msgChan, "UTC", observability.NopLogger{})

	assert.NotNil(t, srv)
	assert.Nil(t, srv.Addr(), "Addr should be nil before ListenAndServe")
}
