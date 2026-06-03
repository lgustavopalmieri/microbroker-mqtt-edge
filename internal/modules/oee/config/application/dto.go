package application

import (
	"encoding/json"
	"fmt"
	"time"

	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

type breakJSON struct {
	StartMin int    `json:"start_minute"`
	EndMin   int    `json:"end_minute"`
	Type     string `json:"type"`
}

type shiftJSON struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	MachineID string      `json:"machine_id"`
	StartMin  int         `json:"start_minute"`
	EndMin    int         `json:"end_minute"`
	Weekdays  []int       `json:"weekdays"`
	TZ        string      `json:"timezone"`
	Breaks    []breakJSON `json:"breaks"`
}

// LoadShiftsFromJSON parses a JSON byte slice into a slice of Shift domain objects.
func LoadShiftsFromJSON(data []byte) ([]ooedomain.Shift, error) {
	var raw []shiftJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("config: invalid shifts JSON: %w", err)
	}

	shifts := make([]ooedomain.Shift, 0, len(raw))
	for _, r := range raw {
		weekdays := make([]time.Weekday, len(r.Weekdays))
		for i, wd := range r.Weekdays {
			weekdays[i] = time.Weekday(wd)
		}

		breaks := make([]ooedomain.Break, len(r.Breaks))
		for i, b := range r.Breaks {
			breaks[i] = ooedomain.Break{
				StartMin: b.StartMin,
				EndMin:   b.EndMin,
				Type:     b.Type,
			}
		}

		shifts = append(shifts, ooedomain.Shift{
			ID:        r.ID,
			Name:      r.Name,
			MachineID: r.MachineID,
			StartMin:  r.StartMin,
			EndMin:    r.EndMin,
			Weekdays:  weekdays,
			TZ:        r.TZ,
			Breaks:    breaks,
		})
	}

	return shifts, nil
}
