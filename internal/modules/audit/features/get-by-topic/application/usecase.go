package application

import "context"

// Execute retrieves all audit records for a given topic.
func (uc *UseCase) Execute(ctx context.Context, topic string) (*Output, error) {
	records, err := uc.repo.GetByTopic(ctx, topic)
	if err != nil {
		uc.logger.Error("audit query failed", "topic", topic, "error", err)
		return nil, err
	}

	return &Output{Records: records}, nil
}
