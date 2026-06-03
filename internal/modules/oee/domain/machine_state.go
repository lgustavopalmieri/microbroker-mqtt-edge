package domain

type MachineState string

const (
	Running     MachineState = "running"
	Stopped     MachineState = "stopped"
	Setup       MachineState = "setup"
	Idle        MachineState = "idle"
	Maintenance MachineState = "maintenance"
	Off         MachineState = "off"
)

func (s MachineState) IsDowntime() bool {
	return s == Stopped || s == Setup || s == Maintenance
}

func (s MachineState) IsPlannedStop() bool {
	return s == Setup || s == Maintenance
}

func (s MachineState) Valid() bool {
	switch s {
	case Running, Stopped, Setup, Idle, Maintenance, Off:
		return true
	}
	return false
}
