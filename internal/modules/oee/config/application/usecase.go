package application

import (
	"context"

	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

// Seed upserts each shift into the store.
func (uc *UseCase) Seed(ctx context.Context, shifts []ooedomain.Shift) error {
	for _, s := range shifts {
		if err := uc.store.Upsert(ctx, s); err != nil {
			uc.logger.Error("config: failed to upsert shift", "shift_id", s.ID, "error", err)
			return err
		}
	}
	return nil
}
