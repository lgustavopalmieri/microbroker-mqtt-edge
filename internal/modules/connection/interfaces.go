package connection

import (
	"microbroker-mqtt-edge/internal/common/observability"
	authdomain "microbroker-mqtt-edge/internal/modules/auth/domain"
)

// Authenticator is the port used by the connection module to validate credentials.
// The implementation is provided by the auth module and injected via constructor.
type Authenticator = authdomain.Authenticator

// Logger is the observability contract used by the connection module.
type Logger = observability.Logger
