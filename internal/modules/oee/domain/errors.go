package domain

import "errors"

var (
	ErrInvalidState  = errors.New("oee: invalid machine state")
	ErrInvalidWindow = errors.New("oee: window From must be before To")
	ErrInvalidShift  = errors.New("oee: shift start_minute must be less than end_minute")
)
