package application

// UseCase applies state transitions to persistent intervals.
type UseCase struct {
	store    IntervalStore
	observer StateObserver
	logger   Logger
}

// NewUseCase creates an ingest-state UseCase with the given dependencies.
func NewUseCase(store IntervalStore, observer StateObserver, logger Logger) *UseCase {
	return &UseCase{store: store, observer: observer, logger: logger}
}
