package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"microbroker-mqtt-edge/internal/common/observability"
	"microbroker-mqtt-edge/internal/modules/oee/config/application"
	"microbroker-mqtt-edge/internal/modules/oee/config/application/mocks"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

func shiftFactory(overrides ...func(*ooedomain.Shift)) ooedomain.Shift {
	s := ooedomain.Shift{
		ID:        "s1",
		Name:      "Manhã",
		MachineID: "*",
		StartMin:  480,
		EndMin:    960,
		TZ:        "UTC",
	}
	for _, o := range overrides {
		o(&s)
	}
	return s
}

func TestSeedUseCase_Seed(t *testing.T) {
	storeErr := errors.New("store: write failed")

	tests := []struct {
		name        string
		shifts      []ooedomain.Shift
		setupMocks  func(store *mocks.MockShiftStore)
		expectError bool
	}{
		{
			name: "happy path - seeds all shifts when store accepts each upsert",
			shifts: []ooedomain.Shift{
				shiftFactory(func(s *ooedomain.Shift) { s.ID = "s1" }),
				shiftFactory(func(s *ooedomain.Shift) { s.ID = "s2" }),
			},
			setupMocks: func(store *mocks.MockShiftStore) {
				store.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(2)
			},
			expectError: false,
		},
		{
			name:   "happy path - empty slice is a no-op, Upsert never called",
			shifts: []ooedomain.Shift{},
			setupMocks: func(store *mocks.MockShiftStore) {
				store.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)
			},
			expectError: false,
		},
		{
			name: "failure - returns store error and stops after first failed upsert",
			shifts: []ooedomain.Shift{
				shiftFactory(func(s *ooedomain.Shift) { s.ID = "s1" }),
				shiftFactory(func(s *ooedomain.Shift) { s.ID = "s2" }),
			},
			setupMocks: func(store *mocks.MockShiftStore) {
				store.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(storeErr).Times(1)
			},
			expectError: true,
		},
		{
			name: "failure - does not call Upsert for remaining shifts after first error",
			shifts: []ooedomain.Shift{
				shiftFactory(func(s *ooedomain.Shift) { s.ID = "fail" }),
				shiftFactory(func(s *ooedomain.Shift) { s.ID = "skipped" }),
				shiftFactory(func(s *ooedomain.Shift) { s.ID = "also-skipped" }),
			},
			setupMocks: func(store *mocks.MockShiftStore) {
				store.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(storeErr).Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mocks.NewMockShiftStore(ctrl)
			tc.setupMocks(store)

			uc := application.NewUseCase(store, observability.NewNopLogger())
			err := uc.Seed(context.Background(), tc.shifts)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
