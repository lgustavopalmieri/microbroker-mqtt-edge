package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvAuthenticator(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		want     bool
	}{
		{"correct credentials", "admin", "secret", true},
		{"wrong username", "wrong", "secret", false},
		{"wrong password", "admin", "wrong", false},
		{"both wrong", "wrong", "wrong", false},
		{"empty username", "", "secret", false},
		{"empty password", "admin", "", false},
		{"both empty", "", "", false},
	}

	auth := NewEnvAuthenticator("admin", "secret")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.Authenticate(tt.username, tt.password)
			assert.Equal(t, tt.want, got)
		})
	}
}
