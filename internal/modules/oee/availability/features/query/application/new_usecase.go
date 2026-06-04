package application

// UseCase computes on-demand availability for a machine over an arbitrary window.
type UseCase struct {
	intervals IntervalReader
	shifts    ShiftReader
	logger    Logger
}

// NewUseCase creates a query UseCase with the given dependencies.
func NewUseCase(intervals IntervalReader, shifts ShiftReader, logger Logger) *UseCase {
	return &UseCase{intervals: intervals, shifts: shifts, logger: logger}
}
