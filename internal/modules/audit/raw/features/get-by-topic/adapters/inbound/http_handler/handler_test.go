package http_handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"microbroker-mqtt-edge/internal/common/observability"
	"microbroker-mqtt-edge/internal/modules/audit/raw/domain"
	"microbroker-mqtt-edge/internal/modules/audit/raw/features/get-by-topic/adapters/inbound/http_handler"
	"microbroker-mqtt-edge/internal/modules/audit/raw/features/get-by-topic/application"
	"microbroker-mqtt-edge/internal/modules/audit/raw/features/get-by-topic/application/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupHandler(t *testing.T) (*http.ServeMux, *mocks.MockRepository, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepository(ctrl)
	uc := application.NewUseCase(repo, observability.NopLogger{})
	h := http_handler.NewHandler(uc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux, repo, ctrl
}

func TestGetByTopic(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setupMocks func(r *mocks.MockRepository)
		wantStatus int
		wantBody   func(t *testing.T, body []byte)
	}{
		{
			name:       "empty topic returns 400",
			path:       "/audit/",
			setupMocks: func(r *mocks.MockRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "returns records for valid topic",
			path: "/audit/machine/status",
			setupMocks: func(r *mocks.MockRepository) {
				r.EXPECT().GetByTopic(gomock.Any(), "machine/status").
					Return([]domain.Record{
						{ClientID: "c1", Topic: "machine/status", Timezone: "UTC", Timestamp: "2026-01-01T00:00:00Z", Payload: `{"status":"running"}`},
					}, nil).Times(1)
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body []byte) {
				var records []domain.Record
				require.NoError(t, json.Unmarshal(body, &records))
				assert.Len(t, records, 1)
				assert.Equal(t, "c1", records[0].ClientID)
				assert.Equal(t, "machine/status", records[0].Topic)
			},
		},
		{
			name: "empty result returns empty array",
			path: "/audit/machine/unknown",
			setupMocks: func(r *mocks.MockRepository) {
				r.EXPECT().GetByTopic(gomock.Any(), "machine/unknown").
					Return(nil, nil).Times(1)
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body []byte) {
				var records []domain.Record
				require.NoError(t, json.Unmarshal(body, &records))
				assert.Empty(t, records)
			},
		},
		{
			name: "repository error returns 500",
			path: "/audit/machine/status",
			setupMocks: func(r *mocks.MockRepository) {
				r.EXPECT().GetByTopic(gomock.Any(), "machine/status").
					Return(nil, errors.New("db down")).Times(1)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, repo, ctrl := setupHandler(t)
			defer ctrl.Finish()

			tt.setupMocks(repo)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantBody != nil {
				tt.wantBody(t, rec.Body.Bytes())
			}
		})
	}
}
