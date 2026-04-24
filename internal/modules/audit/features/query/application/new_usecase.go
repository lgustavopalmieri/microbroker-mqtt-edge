package application

// UseCase orchestrates audit query operations.
type UseCase struct {
	repo   Repository
	logger Logger
}

// NewUseCase creates a query UseCase with the given dependencies.
func NewUseCase(repo Repository, logger Logger) *UseCase {
	return &UseCase{
		repo:   repo,
		logger: logger,
	}
}
