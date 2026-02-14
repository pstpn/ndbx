package httpserver_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"ndbx/pkg/httpserver"
)

func TestHeartbeatMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		endpoint       string
		expectedStatus int
		expectedBody   []byte
		shouldCallNext bool
	}{
		{
			name:           "success /health",
			method:         http.MethodGet,
			endpoint:       "/health",
			expectedStatus: http.StatusOK,
			expectedBody:   []byte(`{"status":"ok"}`),
			shouldCallNext: false,
		},
		{
			name:           "success /health (using HEAD)",
			method:         http.MethodHead,
			endpoint:       "/health",
			expectedStatus: http.StatusOK,
			expectedBody:   []byte(`{"status":"ok"}`),
			shouldCallNext: false,
		},
		{
			name:           "success POST with next call",
			method:         http.MethodPost,
			endpoint:       "/health",
			expectedStatus: http.StatusOK,
			expectedBody:   []byte(`next handler`),
			shouldCallNext: true,
		},
		{
			name:           "success with next handler on different endpoint",
			method:         http.MethodGet,
			endpoint:       "/healthcheck",
			expectedStatus: http.StatusOK,
			expectedBody:   []byte(`next handler`),
			shouldCallNext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nextHandlerCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextHandlerCalled = true
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("next handler"))
			})

			middleware := httpserver.HeartbeatMiddleware("/health")
			handler := middleware(nextHandler)

			req := httptest.NewRequest(tt.method, tt.endpoint, http.NoBody)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			require.Equal(t, tt.shouldCallNext, nextHandlerCalled)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, tt.expectedBody, body)
		})
	}
}
