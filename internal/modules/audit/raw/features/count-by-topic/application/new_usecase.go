package application

// UseCase orchestrates the count-by-topic audit query.
type UseCase struct {
	repo   Repository
	logger Logger
}

// NewUseCase creates a count-by-topic UseCase with the given dependencies.
func NewUseCase(repo Repository, logger Logger) *UseCase {
	return &UseCase{repo: repo, logger: logger}
}
