package audit_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"microbroker-mqtt-edge/internal/common/audit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockReader struct {
	records []audit.Record
	err     error
}

func (m *mockReader) GetByTopic(_ context.Context, _ string) ([]audit.Record, error) {
	return m.records, m.err
}

type nopLogger struct{}

func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}
func (nopLogger) Warn(string, ...any)  {}

// --- tests ---

func TestGetByTopic(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		reader     *mockReader
		wantStatus int
		wantBody   func(t *testing.T, body []byte)
	}{
		{
			name:       "empty topic returns 400",
			path:       "/audit/",
			reader:     &mockReader{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "returns records for valid topic",
			path: "/audit/machine/status",
			reader: &mockReader{
				records: []audit.Record{
					{ClientID: "c1", Topic: "machine/status", Timezone: "UTC", Timestamp: "2026-01-01T00:00:00Z", Payload: `{"status":"running"}`},
				},
			},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body []byte) {
				var records []audit.Record
				require.NoError(t, json.Unmarshal(body, &records))
				assert.Len(t, records, 1)
				assert.Equal(t, "c1", records[0].ClientID)
				assert.Equal(t, "machine/status", records[0].Topic)
			},
		},
		{
			name:       "empty result returns empty array",
			path:       "/audit/machine/unknown",
			reader:     &mockReader{records: nil},
			wantStatus: http.StatusOK,
			wantBody: func(t *testing.T, body []byte) {
				var records []audit.Record
				require.NoError(t, json.Unmarshal(body, &records))
				assert.Empty(t, records)
			},
		},
		{
			name:       "reader error returns 500",
			path:       "/audit/machine/status",
			reader:     &mockReader{err: errors.New("db down")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := audit.NewHandler(tt.reader, nopLogger{})
			mux := http.NewServeMux()
			handler.RegisterRoutes(mux)

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
