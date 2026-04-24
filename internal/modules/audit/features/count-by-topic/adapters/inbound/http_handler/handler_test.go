package http_handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"microbroker-mqtt-edge/internal/common/observability"
	"microbroker-mqtt-edge/internal/modules/audit/features/count-by-topic/adapters/inbound/http_handler"
	"microbroker-mqtt-edge/internal/modules/audit/features/count-by-topic/application"
	"microbroker-mqtt-edge/internal/modules/audit/features/count-by-topic/application/mocks"

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

func TestCountByTopic(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setupMocks func(r *mocks.MockRepository)
		wantStatus int
		wantBody   func(t *testing.T, body []byte)
	}{
		{
			name:       "empty topic returns 400",
			path:       "/audit-count/",
			setupMocks: func(r *mocks.MockRepository) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "returns count for valid topic",
			path: "/audit-count/machine/status",
			setupMocks: func(r *mocks.MockRepository) {
				r.EXPECT().CountByTopic(gomock.Any(), "machine/status").
					Return(int64(42), nil).Times(1)
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body []byte) {
				var result map[string]int64
				require.NoError(t, json.Unmarshal(body, &result))
				assert.Equal(t, int64(42), result["count"])
			},
		},
		{
			name: "repository error returns 500",
			path: "/audit-count/machine/status",
			setupMocks: func(r *mocks.MockRepository) {
				r.EXPECT().CountByTopic(gomock.Any(), "machine/status").
					Return(int64(0), errors.New("db down")).Times(1)
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
