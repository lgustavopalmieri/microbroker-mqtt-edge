package domain

import "errors"

var (
	// ErrAuthFailed indicates authentication failed (bad username or password).
	ErrAuthFailed = errors.New("auth: authentication failed")

	// ErrEmptyCredentials indicates empty username or password was provided.
	ErrEmptyCredentials = errors.New("auth: empty credentials")
)
