package auth

import "microbroker-mqtt-edge/internal/modules/auth/domain"

// EnvAuthenticator validates credentials against configured values.
type EnvAuthenticator struct {
	username string
	password string
}

// NewEnvAuthenticator creates an authenticator with the given credentials.
func NewEnvAuthenticator(username, password string) *EnvAuthenticator {
	return &EnvAuthenticator{
		username: username,
		password: password,
	}
}

// Authenticate returns true only if both username and password match exactly.
func (a *EnvAuthenticator) Authenticate(username, password string) bool {
	if username == "" || password == "" {
		return false
	}
	return username == a.username && password == a.password
}

// Compile-time check that EnvAuthenticator implements Authenticator.
var _ domain.Authenticator = (*EnvAuthenticator)(nil)
