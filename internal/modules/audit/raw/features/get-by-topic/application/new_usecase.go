package application

// UseCase orchestrates the get-by-topic audit query.
type UseCase struct {
	repo   Repository
	logger Logger
}

// NewUseCase creates a get-by-topic UseCase with the given dependencies.
func NewUseCase(repo Repository, logger Logger) *UseCase {
	return &UseCase{repo: repo, logger: logger}
}
