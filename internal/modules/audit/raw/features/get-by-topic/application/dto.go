package application

import "microbroker-mqtt-edge/internal/modules/audit/raw/domain"

// Output is the result of a GetByTopic query.
type Output struct {
	Records []domain.Record
}
