package domain_test

import (
	"testing"

	"microbroker-mqtt-edge/internal/modules/oee/domain"

	"github.com/stretchr/testify/assert"
)

func TestMachineState_IsDowntime(t *testing.T) {
	tests := []struct {
		name  string
		state domain.MachineState
		want  bool
	}{
		{"running is not downtime", domain.Running, false},
		{"stopped is downtime", domain.Stopped, true},
		{"setup is downtime", domain.Setup, true},
		{"idle is not downtime", domain.Idle, false},
		{"maintenance is downtime", domain.Maintenance, true},
		{"off is not downtime", domain.Off, false},
		{"unknown value is not downtime (defensive)", domain.MachineState("unknown"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.state.IsDowntime())
		})
	}
}

func TestMachineState_IsPlannedStop(t *testing.T) {
	tests := []struct {
		name  string
		state domain.MachineState
		want  bool
	}{
		{"setup is a planned stop", domain.Setup, true},
		{"maintenance is a planned stop", domain.Maintenance, true},
		{"stopped is not a planned stop", domain.Stopped, false},
		{"running is not a planned stop", domain.Running, false},
		{"idle is not a planned stop", domain.Idle, false},
		{"off is not a planned stop", domain.Off, false},
		{"unknown value is not a planned stop (defensive)", domain.MachineState("unknown"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.state.IsPlannedStop())
		})
	}
}

func TestMachineState_Valid(t *testing.T) {
	tests := []struct {
		name  string
		state domain.MachineState
		want  bool
	}{
		{"running is valid", domain.Running, true},
		{"stopped is valid", domain.Stopped, true},
		{"setup is valid", domain.Setup, true},
		{"idle is valid", domain.Idle, true},
		{"maintenance is valid", domain.Maintenance, true},
		{"off is valid", domain.Off, true},
		{"empty string is invalid", domain.MachineState(""), false},
		{"unknown literal is invalid", domain.MachineState("paused"), false},
		{"uppercase RUNNING is invalid (case-sensitive)", domain.MachineState("RUNNING"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.state.Valid())
		})
	}
}
