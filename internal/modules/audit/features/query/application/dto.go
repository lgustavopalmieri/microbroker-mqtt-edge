package application

import "microbroker-mqtt-edge/internal/modules/audit/domain"

// GetByTopicOutput is the result of a GetByTopic query.
type GetByTopicOutput struct {
	Records []domain.Record
}

// CountByTopicOutput is the result of a CountByTopic query.
type CountByTopicOutput struct {
	Count int64
}
