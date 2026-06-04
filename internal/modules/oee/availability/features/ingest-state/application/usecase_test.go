package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/application"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/ingest-state/application/mocks"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

var (
	t0 = time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	t1 = t0.Add(time.Hour)

	storeErr = errors.New("store: write failed")

	openInterval = avdomain.StateInterval{
		MachineID: "CNC-01",
		State:     ooedomain.Running,
		StartedAt: t0,
	}
)

func transitionFactory(overrides ...func(*avdomain.StateTransition)) avdomain.StateTransition {
	tr := avdomain.StateTransition{
		MachineID: "CNC-01",
		State:     ooedomain.Stopped,
		Timestamp: t1,
	}
	for _, o := range overrides {
		o(&tr)
	}
	return tr
}

func TestIngestStateUseCase_Apply(t *testing.T) {
	tests := []struct {
		name        string
		transition  avdomain.StateTransition
		setupMocks  func(store *mocks.MockIntervalStore, obs *mocks.MockStateObserver)
		expectError bool
	}{
		{
			name:       "happy path - first event for machine (no open interval) opens interval and notifies observer",
			transition: transitionFactory(),
			setupMocks: func(store *mocks.MockIntervalStore, obs *mocks.MockStateObserver) {
				store.EXPECT().LastOpen(gomock.Any(), "CNC-01").Return(avdomain.StateInterval{}, false, nil).Times(1)
				store.EXPECT().CloseOpen(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().OpenInterval(gomock.Any(), "CNC-01", ooedomain.Stopped, t1).Return(nil).Times(1)
				obs.EXPECT().Apply(gomock.Any()).Times(1)
			},
		},
		{
			name:       "happy path - valid transition closes open interval, opens new one, notifies observer",
			transition: transitionFactory(),
			setupMocks: func(store *mocks.MockIntervalStore, obs *mocks.MockStateObserver) {
				store.EXPECT().LastOpen(gomock.Any(), "CNC-01").Return(openInterval, true, nil).Times(1)
				store.EXPECT().CloseOpen(gomock.Any(), "CNC-01", t1).Return(nil).Times(1)
				store.EXPECT().OpenInterval(gomock.Any(), "CNC-01", ooedomain.Stopped, t1).Return(nil).Times(1)
				obs.EXPECT().Apply(gomock.Any()).Times(1)
			},
		},
		{
			name: "business rule - no-op transition (same state as open) is ignored: no store writes, observer not called",
			transition: transitionFactory(func(tr *avdomain.StateTransition) {
				tr.State = ooedomain.Running
			}),
			setupMocks: func(store *mocks.MockIntervalStore, obs *mocks.MockStateObserver) {
				store.EXPECT().LastOpen(gomock.Any(), "CNC-01").Return(openInterval, true, nil).Times(1)
				store.EXPECT().CloseOpen(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().OpenInterval(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				obs.EXPECT().Apply(gomock.Any()).Times(0)
			},
		},
		{
			name: "business rule - non-increasing timestamp is ignored: no store writes, observer not called",
			transition: transitionFactory(func(tr *avdomain.StateTransition) {
				tr.Timestamp = t0
			}),
			setupMocks: func(store *mocks.MockIntervalStore, obs *mocks.MockStateObserver) {
				store.EXPECT().LastOpen(gomock.Any(), "CNC-01").Return(openInterval, true, nil).Times(1)
				store.EXPECT().CloseOpen(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().OpenInterval(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				obs.EXPECT().Apply(gomock.Any()).Times(0)
			},
		},
		{
			name:       "failure - LastOpen error is propagated, no further store calls",
			transition: transitionFactory(),
			setupMocks: func(store *mocks.MockIntervalStore, obs *mocks.MockStateObserver) {
				store.EXPECT().LastOpen(gomock.Any(), "CNC-01").Return(avdomain.StateInterval{}, false, storeErr).Times(1)
				store.EXPECT().CloseOpen(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().OpenInterval(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				obs.EXPECT().Apply(gomock.Any()).Times(0)
			},
			expectError: true,
		},
		{
			name:       "failure - CloseOpen error is propagated, OpenInterval and observer not called",
			transition: transitionFactory(),
			setupMocks: func(store *mocks.MockIntervalStore, obs *mocks.MockStateObserver) {
				store.EXPECT().LastOpen(gomock.Any(), "CNC-01").Return(openInterval, true, nil).Times(1)
				store.EXPECT().CloseOpen(gomock.Any(), "CNC-01", t1).Return(storeErr).Times(1)
				store.EXPECT().OpenInterval(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				obs.EXPECT().Apply(gomock.Any()).Times(0)
			},
			expectError: true,
		},
		{
			name:       "failure - OpenInterval error is propagated, observer not called",
			transition: transitionFactory(),
			setupMocks: func(store *mocks.MockIntervalStore, obs *mocks.MockStateObserver) {
				store.EXPECT().LastOpen(gomock.Any(), "CNC-01").Return(avdomain.StateInterval{}, false, nil).Times(1)
				store.EXPECT().OpenInterval(gomock.Any(), "CNC-01", ooedomain.Stopped, t1).Return(storeErr).Times(1)
				obs.EXPECT().Apply(gomock.Any()).Times(0)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mocks.NewMockIntervalStore(ctrl)
			obs := mocks.NewMockStateObserver(ctrl)
			tc.setupMocks(store, obs)

			uc := application.NewUseCase(store, obs, observability.NewNopLogger())
			err := uc.Apply(context.Background(), tc.transition)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
