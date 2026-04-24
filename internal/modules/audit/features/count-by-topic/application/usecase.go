package application

import "context"

// Execute returns the number of audit records for a given topic.
func (uc *UseCase) Execute(ctx context.Context, topic string) (*Output, error) {
	count, err := uc.repo.CountByTopic(ctx, topic)
	if err != nil {
		uc.logger.Error("audit count failed", "topic", topic, "error", err)
		return nil, err
	}

	return &Output{Count: count}, nil
}
