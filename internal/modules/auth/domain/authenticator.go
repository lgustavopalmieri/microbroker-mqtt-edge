package domain

// Authenticator validates client credentials.
// Implementations must be safe for concurrent use.
type Authenticator interface {
	Authenticate(username, password string) bool
}
