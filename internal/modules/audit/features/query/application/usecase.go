package application

import "context"

// GetByTopic retrieves all audit records for a given topic.
func (uc *UseCase) GetByTopic(ctx context.Context, topic string) (*GetByTopicOutput, error) {
	records, err := uc.repo.GetByTopic(ctx, topic)
	if err != nil {
		uc.logger.Error("audit query failed", "topic", topic, "error", err)
		return nil, err
	}

	return &GetByTopicOutput{Records: records}, nil
}

// CountByTopic returns the number of audit records for a given topic.
func (uc *UseCase) CountByTopic(ctx context.Context, topic string) (*CountByTopicOutput, error) {
	count, err := uc.repo.CountByTopic(ctx, topic)
	if err != nil {
		uc.logger.Error("audit count failed", "topic", topic, "error", err)
		return nil, err
	}

	return &CountByTopicOutput{Count: count}, nil
}
