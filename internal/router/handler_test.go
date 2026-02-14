package router_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"ndbx/internal/router"
	oas "ndbx/internal/router/ogen"
	"ndbx/pkg/logger"
)

func TestHandler_APIPing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   []byte
	}{
		{
			name:           "successful ping",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody:   []byte(`pong`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, "/api/ping", http.NoBody)
			w := httptest.NewRecorder()

			srv, err := oas.NewServer(router.NewHandler(logger.NewWithOutput("debug", io.Discard)))
			require.NoError(t, err)
			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, tt.expectedBody, body)
		})
	}
}
