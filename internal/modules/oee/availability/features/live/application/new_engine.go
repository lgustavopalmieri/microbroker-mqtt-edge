package application

import (
	"sync"
	"time"

	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
)

// machineData holds the in-memory accumulator for one machine.
type machineData struct {
	intervals   []avdomain.StateInterval
	windowStart time.Time
}

// Engine accumulates per-machine state transitions and emits availability snapshots on each tick.
type Engine struct {
	mu           sync.Mutex
	machines     map[string]*machineData
	sink         AvailabilitySink
	shifts       ShiftReader
	intervals    StateIntervalReader
	logger       Logger
	tickInterval time.Duration
	now          func() time.Time
}

// NewEngine creates a live availability Engine.
// now is injectable to allow deterministic tests (pass time.Now in production).
func NewEngine(
	sink AvailabilitySink,
	shifts ShiftReader,
	intervals StateIntervalReader,
	logger Logger,
	tickInterval time.Duration,
	now func() time.Time,
) *Engine {
	return &Engine{
		machines:     make(map[string]*machineData),
		sink:         sink,
		shifts:       shifts,
		intervals:    intervals,
		logger:       logger,
		tickInterval: tickInterval,
		now:          now,
	}
}
