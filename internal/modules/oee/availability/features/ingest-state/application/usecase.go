package application

import "context"

// Apply persists a state transition as a closed+opened interval pair and notifies the observer.
// No-op transitions and non-increasing timestamps are silently ignored.
func (uc *UseCase) Apply(ctx context.Context, t StateTransition) error {
	last, hasOpen, err := uc.store.LastOpen(ctx, t.MachineID)
	if err != nil {
		uc.logger.Error("ingest-state: failed to read last open interval", "machine_id", t.MachineID, "error", err)
		return err
	}

	if hasOpen {
		if last.State == t.State {
			return nil
		}
		if !t.Timestamp.After(last.StartedAt) {
			return nil
		}
		if err := uc.store.CloseOpen(ctx, t.MachineID, t.Timestamp); err != nil {
			uc.logger.Error("ingest-state: failed to close open interval", "machine_id", t.MachineID, "error", err)
			return err
		}
	}

	if err := uc.store.OpenInterval(ctx, t.MachineID, t.State, t.Timestamp); err != nil {
		uc.logger.Error("ingest-state: failed to open new interval", "machine_id", t.MachineID, "error", err)
		return err
	}

	uc.observer.Apply(t)
	return nil
}
