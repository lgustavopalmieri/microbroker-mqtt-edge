package http_handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"microbroker-mqtt-edge/internal/common/observability"
	avdomain "microbroker-mqtt-edge/internal/modules/oee/availability/domain"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/query/adapters/inbound/http_handler"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/query/application"
	"microbroker-mqtt-edge/internal/modules/oee/availability/features/query/application/mocks"
	ooedomain "microbroker-mqtt-edge/internal/modules/oee/domain"
)

func setupHandler(t *testing.T) (*http.ServeMux, *mocks.MockIntervalReader, *mocks.MockShiftReader, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)
	intervals := mocks.NewMockIntervalReader(ctrl)
	shifts := mocks.NewMockShiftReader(ctrl)
	uc := application.NewUseCase(intervals, shifts, observability.NopLogger{})
	h := http_handler.NewHandler(uc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux, intervals, shifts, ctrl
}

const (
	validFrom = "2026-06-01T08:00:00Z"
	validTo   = "2026-06-01T12:00:00Z"
	basePath  = "/availability/m1"
)

var (
	fromTime = time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	toTime   = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
)

func noMocks(intervals *mocks.MockIntervalReader, shifts *mocks.MockShiftReader) {
	intervals.EXPECT().ByMachineRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	shifts.EXPECT().ForMachineWindow(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func TestAvailabilityHandler_GetAvailability(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		setupMocks func(*mocks.MockIntervalReader, *mocks.MockShiftReader)
		wantStatus int
		wantBody   func(t *testing.T, body []byte)
	}{
		{
			name:       "validation - missing from param returns 400",
			url:        basePath + "?to=" + validTo,
			setupMocks: noMocks,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "validation - unparseable from timestamp returns 400",
			url:        basePath + "?from=not-a-date&to=" + validTo,
			setupMocks: noMocks,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "validation - from equal to to returns 400",
			url:        basePath + "?from=" + validFrom + "&to=" + validFrom,
			setupMocks: noMocks,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "happy path - valid params and use case succeeds returns 200 with JSON output",
			url:  basePath + "?from=" + validFrom + "&to=" + validTo,
			setupMocks: func(intervals *mocks.MockIntervalReader, shifts *mocks.MockShiftReader) {
				end := toTime
				intervals.EXPECT().
					ByMachineRange(gomock.Any(), "m1", fromTime, toTime).
					Times(1).
					Return([]avdomain.StateInterval{
						{MachineID: "m1", State: ooedomain.Running, StartedAt: fromTime, EndedAt: &end},
					}, nil)
				shifts.EXPECT().
					ForMachineWindow(gomock.Any(), "m1", gomock.Any()).
					Times(1).
					Return(4*time.Hour, true, nil)
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body []byte) {
				var out application.Output
				require.NoError(t, json.Unmarshal(body, &out))
				assert.Equal(t, "m1", out.MachineID)
				assert.True(t, out.HasData)
				assert.False(t, out.Flags.NoData)
			},
		},
		{
			name: "business rule - no-data window returns 200 with no_data flag true in body",
			url:  basePath + "?from=" + validFrom + "&to=" + validTo,
			setupMocks: func(intervals *mocks.MockIntervalReader, shifts *mocks.MockShiftReader) {
				intervals.EXPECT().
					ByMachineRange(gomock.Any(), "m1", fromTime, toTime).
					Times(1).
					Return([]avdomain.StateInterval{}, nil)
				shifts.EXPECT().
					ForMachineWindow(gomock.Any(), "m1", gomock.Any()).
					Times(1).
					Return(4*time.Hour, true, nil)
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body []byte) {
				var out application.Output
				require.NoError(t, json.Unmarshal(body, &out))
				assert.True(t, out.Flags.NoData)
				assert.False(t, out.HasData)
			},
		},
		{
			name: "failure - use case error returns 500",
			url:  basePath + "?from=" + validFrom + "&to=" + validTo,
			setupMocks: func(intervals *mocks.MockIntervalReader, shifts *mocks.MockShiftReader) {
				intervals.EXPECT().
					ByMachineRange(gomock.Any(), "m1", fromTime, toTime).
					Times(1).
					Return(nil, errors.New("db down"))
				shifts.EXPECT().ForMachineWindow(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, intervals, shifts, ctrl := setupHandler(t)
			defer ctrl.Finish()

			tt.setupMocks(intervals, shifts)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantBody != nil {
				tt.wantBody(t, rec.Body.Bytes())
			}
		})
	}
}
