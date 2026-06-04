package domain

import (
	"time"

	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

type StateInterval struct {
	MachineID string
	State     ooedomain.MachineState
	StartedAt time.Time
	EndedAt   *time.Time
	Reason    string
}

func (s StateInterval) IsOpen() bool {
	return s.EndedAt == nil
}
