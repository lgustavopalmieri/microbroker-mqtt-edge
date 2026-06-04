package domain

import (
	"time"

	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

// StateTransition carries a validated machine-state change event.
type StateTransition struct {
	MachineID     string
	State         ooedomain.MachineState
	PreviousState ooedomain.MachineState
	Reason        string
	Timestamp     time.Time
}
