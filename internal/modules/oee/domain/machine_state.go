package domain

// MachineState represents the operational state of a machine.
type MachineState string

const (
	Running     MachineState = "running"
	Stopped     MachineState = "stopped"
	Setup       MachineState = "setup"
	Idle        MachineState = "idle"
	Maintenance MachineState = "maintenance"
	Off         MachineState = "off"
)

// IsDowntime reports whether the state counts as downtime (stopped | setup | maintenance).
func (s MachineState) IsDowntime() bool {
	return s == Stopped || s == Setup || s == Maintenance
}

// IsPlannedStop reports whether the state counts as a planned stop (setup | maintenance).
func (s MachineState) IsPlannedStop() bool {
	return s == Setup || s == Maintenance
}

// Valid reports whether s is a recognised machine state.
func (s MachineState) Valid() bool {
	switch s {
	case Running, Stopped, Setup, Idle, Maintenance, Off:
		return true
	}
	return false
}
