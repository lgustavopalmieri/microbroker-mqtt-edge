package application

// UseCase seeds shift configuration into the store.
type UseCase struct {
	store  ShiftStore
	logger Logger
}

// NewUseCase creates a config seed UseCase with the given dependencies.
func NewUseCase(store ShiftStore, logger Logger) *UseCase {
	return &UseCase{store: store, logger: logger}
}
