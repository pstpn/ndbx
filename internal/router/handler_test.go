package router_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/ovechkin-dm/mockio/v2/mockopts"
	"github.com/stretchr/testify/require"

	"ndbx/internal/router"
	oas "ndbx/internal/router/ogen"
	"ndbx/internal/service/dto"
	"ndbx/pkg/logger"
)

func TestHandler_APIHealth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cookie         string
		expectedStatus int
		expectedBody   string
		expectedCookie string
	}{
		{
			name:           "successful health without cookie",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
		{
			name:           "successful health with cookie",
			cookie:         "X-Session-Id=sid-1; foo=bar",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
			expectedCookie: "X-Session-Id=sid-1; foo=bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := mock.NewMockController(t, mockopts.StrictVerify())
			sessionService := mock.Mock[router.SessionService](ctrl)

			req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			w := httptest.NewRecorder()

			srv := newServer(t, sessionService)
			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			require.Equal(t, tt.expectedCookie, resp.Header.Get("Cookie"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.JSONEq(t, tt.expectedBody, string(body))
		})
	}
}

func TestHandler_APISession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		cookie            string
		setup             func(sessionService router.SessionService)
		expectedStatus    int
		expectedSetCookie string
	}{
		{
			name: "create session when cookie missing",
			setup: func(sessionService router.SessionService) {
				mock.WhenDouble(sessionService.CreateSession(mock.AnyContext())).
					ThenReturn(&dto.CreateSessionResp{SID: "new-sid", MaxAgeSeconds: 100}, nil)
			},
			expectedStatus:    http.StatusCreated,
			expectedSetCookie: "X-Session-Id=new-sid; HttpOnly; Path=/; Max-Age=100",
		},
		{
			name:   "create session when session cookie missing in header",
			cookie: "foo=bar",
			setup: func(sessionService router.SessionService) {
				mock.WhenDouble(sessionService.CreateSession(mock.AnyContext())).
					ThenReturn(&dto.CreateSessionResp{SID: "created-sid", MaxAgeSeconds: 120}, nil)
			},
			expectedStatus:    http.StatusCreated,
			expectedSetCookie: "X-Session-Id=created-sid; HttpOnly; Path=/; Max-Age=120",
		},
		{
			name:   "extend existing session",
			cookie: "a=1; X-Session-Id=existing-sid; b=2",
			setup: func(sessionService router.SessionService) {
				mock.WhenDouble(sessionService.CreateOrExtendSession(
					mock.AnyContext(), mock.Equal(&dto.CreateOrExtendSessionReq{SID: "existing-sid"})),
				).ThenReturn(&dto.CreateOrExtendSessionResp{SID: "existing-sid", MaxAgeSeconds: 90, IsCreated: false}, nil)
			},
			expectedStatus:    http.StatusOK,
			expectedSetCookie: "X-Session-Id=existing-sid; HttpOnly; Path=/; Max-Age=90",
		},
		{
			name:   "create session by create-or-extend when session was recreated",
			cookie: "X-Session-Id=expired-sid",
			setup: func(sessionService router.SessionService) {
				mock.WhenDouble(sessionService.CreateOrExtendSession(
					mock.AnyContext(), mock.Equal(&dto.CreateOrExtendSessionReq{SID: "expired-sid"})),
				).ThenReturn(&dto.CreateOrExtendSessionResp{SID: "newer-sid", MaxAgeSeconds: 60, IsCreated: true}, nil)
			},
			expectedStatus:    http.StatusCreated,
			expectedSetCookie: "X-Session-Id=newer-sid; HttpOnly; Path=/; Max-Age=60",
		},
		{
			name:   "service error returns internal server error",
			cookie: "X-Session-Id=sid-with-error",
			setup: func(sessionService router.SessionService) {
				mock.WhenDouble(sessionService.CreateOrExtendSession(
					mock.AnyContext(), mock.Equal(&dto.CreateOrExtendSessionReq{SID: "sid-with-error"})),
				).ThenReturn(nil, errors.New("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := mock.NewMockController(t, mockopts.StrictVerify())
			sessionService := mock.Mock[router.SessionService](ctrl)
			tt.setup(sessionService)

			req := httptest.NewRequest(http.MethodPost, "/session", http.NoBody)
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			w := httptest.NewRecorder()

			srv := newServer(t, sessionService)
			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			require.Equal(t, tt.expectedSetCookie, resp.Header.Get("Set-Cookie"))
		})
	}
}

func newServer(t *testing.T, sessionService router.SessionService) http.Handler {
	t.Helper()

	srv, err := oas.NewServer(router.NewHandler(logger.NewWithOutput("debug", io.Discard), sessionService))
	require.NoError(t, err)

	return srv
}
