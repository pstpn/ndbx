package router_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/ovechkin-dm/mockio/v2/mockopts"
	"github.com/stretchr/testify/require"

	"ndbx/internal/router"
	oas "ndbx/internal/router/ogen"
	"ndbx/internal/service"
	"ndbx/internal/service/dto"
	"ndbx/pkg/logger"
)

const mockTTL = 10

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
			expectedCookie: fmt.Sprintf("X-Session-Id=sid-1; HttpOnly; Path=/; Max-Age=%d", mockTTL),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := mock.NewMockController(t, mockopts.StrictVerify())
			sessionService := mock.Mock[router.SessionService](ctrl)
			userService := mock.Mock[router.UserService](ctrl)
			eventService := mock.Mock[router.EventService](ctrl)

			req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			w := httptest.NewRecorder()

			h := newHandler(t, sessionService, userService, eventService)
			srv, err := oas.NewServer(h)
			require.NoError(t, err)

			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			require.Equal(t, tt.expectedCookie, resp.Header.Get("Set-Cookie"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.JSONEq(t, tt.expectedBody, string(body))
		})
	}
}

func TestHandler_APISession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cookie         string
		setup          func(h *router.Handler)
		expectedStatus int
		expectedCookie string
	}{
		{
			name: "create session when cookie missing",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.SessionService.CreateSession(mock.AnyContext(), mock.Any[*dto.CreateSessionReq]())).
					ThenReturn(&dto.CreateSessionResp{SID: "new-sid", TTL: time.Second * mockTTL}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedCookie: fmt.Sprintf("X-Session-Id=new-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
		},
		{
			name:   "create session when session cookie missing in header",
			cookie: "foo=bar",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.SessionService.CreateSession(mock.AnyContext(), mock.Any[*dto.CreateSessionReq]())).
					ThenReturn(&dto.CreateSessionResp{SID: "created-sid", TTL: time.Second * mockTTL}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedCookie: fmt.Sprintf("X-Session-Id=created-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
		},
		{
			name:   "extend existing session",
			cookie: "a=1; X-Session-Id=existing-sid; b=2",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.SessionService.CreateOrExtendSession(
					mock.AnyContext(), mock.Equal(&dto.CreateOrExtendSessionReq{SID: "existing-sid"})),
				).ThenReturn(&dto.CreateOrExtendSessionResp{SID: "existing-sid", TTL: time.Second * mockTTL, IsCreated: false}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCookie: fmt.Sprintf("X-Session-Id=existing-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
		},
		{
			name:   "create session by create-or-extend when session was recreated",
			cookie: "X-Session-Id=expired-sid",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.SessionService.CreateOrExtendSession(
					mock.AnyContext(), mock.Equal(&dto.CreateOrExtendSessionReq{SID: "expired-sid"})),
				).ThenReturn(&dto.CreateOrExtendSessionResp{SID: "newer-sid", TTL: time.Second * mockTTL, IsCreated: true}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedCookie: fmt.Sprintf("X-Session-Id=newer-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
		},
		{
			name:   "service error returns internal server error",
			cookie: "X-Session-Id=sid-with-error",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.SessionService.CreateOrExtendSession(
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
			userService := mock.Mock[router.UserService](ctrl)
			eventService := mock.Mock[router.EventService](ctrl)
			h := newHandler(t, sessionService, userService, eventService)
			if tt.setup != nil {
				tt.setup(h)
			}

			req := httptest.NewRequest(http.MethodPost, "/session", http.NoBody)
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			w := httptest.NewRecorder()

			srv, err := oas.NewServer(h)
			require.NoError(t, err)

			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			require.Equal(t, tt.expectedCookie, resp.Header.Get("Set-Cookie"))
		})
	}
}

func TestHandler_APIRegister(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cookie         string
		requestBody    oas.UserRegisterRequest
		setup          func(h *router.Handler)
		expectedStatus int
		expectedCookie string
		expectedBody   string
	}{
		{
			name: "successful registration (without cookie)",
			requestBody: oas.UserRegisterRequest{
				FullName: "John Doe",
				Username: "johndoe",
				Password: "password123",
			},
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.Register(mock.AnyContext(), mock.Equal(&dto.RegisterReq{
					FullName: "John Doe",
					Username: "johndoe",
					Password: "password123",
				}))).ThenReturn(&dto.RegisterResp{ID: "user-id"}, nil)
				mock.WhenDouble(h.SessionService.CreateSession(mock.AnyContext(), mock.Equal(&dto.CreateSessionReq{
					UserID: "user-id",
				}))).ThenReturn(&dto.CreateSessionResp{SID: "new-sid", TTL: time.Second * mockTTL}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedCookie: fmt.Sprintf("X-Session-Id=new-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
		},
		{
			name:   "successful registration (with cookie)",
			cookie: "X-Session-Id=sid",
			requestBody: oas.UserRegisterRequest{
				FullName: "John Doe",
				Username: "johndoe",
				Password: "password123",
			},
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.Register(mock.AnyContext(), mock.Equal(&dto.RegisterReq{
					FullName: "John Doe",
					Username: "johndoe",
					Password: "password123",
				}))).ThenReturn(&dto.RegisterResp{ID: "user-id"}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedCookie: fmt.Sprintf("X-Session-Id=sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
		},
		{
			name:   "user already exists",
			cookie: "X-Session-Id=sid",
			requestBody: oas.UserRegisterRequest{
				FullName: "Jane Doe",
				Username: "janedoe",
				Password: "password123",
			},
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.Register(mock.AnyContext(), mock.Any[*dto.RegisterReq]())).
					ThenReturn(nil, service.ErrUserAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectedCookie: fmt.Sprintf("X-Session-Id=sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
			expectedBody:   `{"message":"user already exists"}`,
		},
		{
			name: "empty full_name",
			requestBody: oas.UserRegisterRequest{
				FullName: "",
				Username: "johndoe",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"message":"invalid \"full_name\" field"}`,
		},
		{
			name: "empty username",
			requestBody: oas.UserRegisterRequest{
				FullName: "John Doe",
				Username: "",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"message":"invalid \"username\" field"}`,
		},
		{
			name: "empty password",
			requestBody: oas.UserRegisterRequest{
				FullName: "John Doe",
				Username: "johndoe",
				Password: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"message":"invalid \"password\" field"}`,
		},
		{
			name: "service error",
			requestBody: oas.UserRegisterRequest{
				FullName: "John Doe",
				Username: "johndoe",
				Password: "password123",
			},
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.Register(mock.AnyContext(), mock.Any[*dto.RegisterReq]())).
					ThenReturn(nil, errors.New("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := mock.NewMockController(t, mockopts.StrictVerify())
			sessionService := mock.Mock[router.SessionService](ctrl)
			userService := mock.Mock[router.UserService](ctrl)
			eventService := mock.Mock[router.EventService](ctrl)
			h := newHandler(t, sessionService, userService, eventService)
			if tt.setup != nil {
				tt.setup(h)
			}

			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			w := httptest.NewRecorder()

			srv, err := oas.NewServer(h)
			require.NoError(t, err)

			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			if tt.expectedCookie != "" {
				require.Equal(t, tt.expectedCookie, resp.Header.Get("Set-Cookie"))
			}
			if tt.expectedBody != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.JSONEq(t, tt.expectedBody, string(body))
			}
		})
	}
}

func TestHandler_APILogin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cookie         string
		requestBody    oas.LoginRequest
		setup          func(h *router.Handler)
		expectedStatus int
		expectedCookie string
		expectedBody   string
	}{
		{
			name: "successful login without existing session",
			requestBody: oas.LoginRequest{
				Username: "johndoe",
				Password: "password123",
			},
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.Authenticate(mock.AnyContext(), mock.Equal(&dto.AuthenticateReq{
					Username: "johndoe",
					Password: "password123",
				}))).ThenReturn(&dto.AuthenticateResp{ID: "user-id"}, nil)
				mock.WhenDouble(h.SessionService.CreateSession(mock.AnyContext(), mock.Equal(&dto.CreateSessionReq{
					UserID: "user-id",
				}))).ThenReturn(&dto.CreateSessionResp{SID: "new-sid", TTL: time.Second * mockTTL}, nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedCookie: "X-Session-Id=new-sid; HttpOnly; Path=/; Max-Age=10",
		},
		{
			name:   "successful login with existing session",
			cookie: "X-Session-Id=existing-sid",
			requestBody: oas.LoginRequest{
				Username: "johndoe",
				Password: "password123",
			},
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.Authenticate(mock.AnyContext(), mock.Equal(&dto.AuthenticateReq{
					Username: "johndoe",
					Password: "password123",
				}))).ThenReturn(&dto.AuthenticateResp{ID: "user-id"}, nil)
				mock.WhenDouble(h.SessionService.CreateOrExtendSession(mock.AnyContext(), mock.Equal(&dto.CreateOrExtendSessionReq{
					SID:    "existing-sid",
					UserID: "user-id",
				}))).ThenReturn(&dto.CreateOrExtendSessionResp{SID: "existing-sid", TTL: time.Second * mockTTL, IsCreated: false}, nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedCookie: "X-Session-Id=existing-sid; HttpOnly; Path=/; Max-Age=10",
		},
		{
			name: "invalid credentials",
			requestBody: oas.LoginRequest{
				Username: "johndoe",
				Password: "wrongpassword",
			},
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.Authenticate(mock.AnyContext(), mock.Any[*dto.AuthenticateReq]())).
					ThenReturn(nil, service.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"message":"invalid credentials"}`,
		},
		{
			name: "empty username",
			requestBody: oas.LoginRequest{
				Username: "",
				Password: "password123",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"message":"invalid credentials"}`,
		},
		{
			name: "empty password",
			requestBody: oas.LoginRequest{
				Username: "johndoe",
				Password: "",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"message":"invalid credentials"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := mock.NewMockController(t, mockopts.StrictVerify())
			sessionService := mock.Mock[router.SessionService](ctrl)
			userService := mock.Mock[router.UserService](ctrl)
			eventService := mock.Mock[router.EventService](ctrl)
			h := newHandler(t, sessionService, userService, eventService)
			if tt.setup != nil {
				tt.setup(h)
			}

			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			w := httptest.NewRecorder()

			srv, err := oas.NewServer(h)
			require.NoError(t, err)

			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			require.Equal(t, tt.expectedCookie, resp.Header.Get("Set-Cookie"))
			if tt.expectedBody != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.JSONEq(t, tt.expectedBody, string(body))
			}
		})
	}
}

func TestHandler_APILogout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cookie         string
		setup          func(h *router.Handler)
		expectedStatus int
		expectedCookie string
		expectedBody   string
	}{
		{
			name:   "successful logout with session",
			cookie: "X-Session-Id=session-to-delete",
			setup: func(h *router.Handler) {
				mock.WhenSingle(h.SessionService.DeleteSession(mock.AnyContext(), mock.Equal(&dto.DeleteSessionReq{
					SID: "session-to-delete",
				}))).ThenReturn(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedCookie: "X-Session-Id=session-to-delete; HttpOnly; Path=/; Max-Age=0",
		},
		{
			name:           "logout without session",
			expectedStatus: http.StatusUnauthorized,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
		},
		{
			name:   "service error on delete",
			cookie: "X-Session-Id=session-error",
			setup: func(h *router.Handler) {
				mock.WhenSingle(h.SessionService.DeleteSession(mock.AnyContext(), mock.Any[*dto.DeleteSessionReq]())).
					ThenReturn(errors.New("delete error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"internal error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := mock.NewMockController(t, mockopts.StrictVerify())
			sessionService := mock.Mock[router.SessionService](ctrl)
			userService := mock.Mock[router.UserService](ctrl)
			eventService := mock.Mock[router.EventService](ctrl)
			h := newHandler(t, sessionService, userService, eventService)
			if tt.setup != nil {
				tt.setup(h)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/logout", http.NoBody)
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			w := httptest.NewRecorder()

			srv, err := oas.NewServer(h)
			require.NoError(t, err)

			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			require.Equal(t, tt.expectedCookie, resp.Header.Get("Set-Cookie"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if tt.expectedBody == "" {
				require.Empty(t, body)
			} else {
				require.JSONEq(t, tt.expectedBody, string(body))
			}
		})
	}
}

func TestHandler_APICreateEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cookie         string
		requestBody    string
		setup          func(h *router.Handler)
		expectedStatus int
		expectedCookie string
		expectedBody   string
	}{
		{
			name:   "successful event creation",
			cookie: "X-Session-Id=valid-sid",
			requestBody: createEventRequestBody(
				"Test Event",
				"Description",
				"Test Address",
				"2023-01-01T10:00:00Z",
				"2023-01-01T12:00:00Z",
			),
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.SessionService.GetSession(mock.AnyContext(), mock.Equal(&dto.GetSessionReq{
					SID: "valid-sid",
				}))).ThenReturn(&dto.GetSessionResp{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					UserID:    "user-id",
				}, nil)
				mock.WhenDouble(h.EventService.CreateEvent(mock.AnyContext(), mock.Any[*dto.CreateEventReq]())).
					ThenReturn(&dto.CreateEventResp{ID: "event-id"}, nil)
				mock.WhenDouble(h.SessionService.CreateOrExtendSession(mock.AnyContext(), mock.Equal(&dto.CreateOrExtendSessionReq{
					SID:    "valid-sid",
					UserID: "user-id",
				}))).ThenReturn(&dto.CreateOrExtendSessionResp{SID: "valid-sid", TTL: time.Second * mockTTL, IsCreated: false}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedCookie: fmt.Sprintf("X-Session-Id=valid-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
			expectedBody:   `{"id":"event-id"}`,
		},
		{
			name:           "missing session",
			requestBody:    createEventRequestBody("Test Event", "", "Test Address", "2023-01-01T10:00:00Z", "2023-01-01T12:00:00Z"),
			expectedStatus: http.StatusUnauthorized,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   "",
		},
		{
			name:        "session not found",
			cookie:      "X-Session-Id=invalid-sid",
			requestBody: createEventRequestBody("Test Event", "", "Test Address", "2023-01-01T10:00:00Z", "2023-01-01T12:00:00Z"),
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.SessionService.GetSession(mock.AnyContext(), mock.Equal(&dto.GetSessionReq{
					SID: "invalid-sid",
				}))).ThenReturn(nil, service.ErrNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCookie: fmt.Sprintf("X-Session-Id=invalid-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
			expectedBody:   "",
		},
		{
			name:        "session without user",
			cookie:      "X-Session-Id=anon-sid",
			requestBody: createEventRequestBody("Test Event", "", "Test Address", "2023-01-01T10:00:00Z", "2023-01-01T12:00:00Z"),
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.SessionService.GetSession(mock.AnyContext(), mock.Equal(&dto.GetSessionReq{
					SID: "anon-sid",
				}))).ThenReturn(&dto.GetSessionResp{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					UserID:    "",
				}, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCookie: fmt.Sprintf("X-Session-Id=anon-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
			expectedBody:   "",
		},
		{
			name:           "empty title",
			cookie:         "X-Session-Id=valid-sid",
			requestBody:    createEventRequestBody("", "", "Test Address", "2023-01-01T10:00:00Z", "2023-01-01T12:00:00Z"),
			expectedStatus: http.StatusBadRequest,
			expectedCookie: fmt.Sprintf("X-Session-Id=valid-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
			expectedBody:   `{"message":"invalid \"title\" field"}`,
		},
		{
			name:           "empty address",
			cookie:         "X-Session-Id=valid-sid",
			requestBody:    createEventRequestBody("Test Event", "", "", "2023-01-01T10:00:00Z", "2023-01-01T12:00:00Z"),
			expectedStatus: http.StatusBadRequest,
			expectedCookie: fmt.Sprintf("X-Session-Id=valid-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
			expectedBody:   `{"message":"invalid \"address\" field"}`,
		},
		{
			name:           "invalid started_at",
			cookie:         "X-Session-Id=valid-sid",
			requestBody:    createEventRequestBody("Test Event", "", "Test Address", "invalid-date", "2023-01-01T12:00:00Z"),
			expectedStatus: http.StatusBadRequest,
			expectedCookie: fmt.Sprintf("X-Session-Id=valid-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
			expectedBody:   `{"message":"invalid \"started_at\" field"}`,
		},
		{
			name:           "invalid finished_at",
			cookie:         "X-Session-Id=valid-sid",
			requestBody:    createEventRequestBody("Test Event", "", "Test Address", "2023-01-01T10:00:00Z", "invalid-date"),
			expectedStatus: http.StatusBadRequest,
			expectedCookie: fmt.Sprintf("X-Session-Id=valid-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
			expectedBody:   `{"message":"invalid \"finished_at\" field"}`,
		},
		{
			name:        "event already exists",
			cookie:      "X-Session-Id=valid-sid",
			requestBody: createEventRequestBody("Test Event", "", "Test Address", "2023-01-01T10:00:00Z", "2023-01-01T12:00:00Z"),
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.SessionService.GetSession(mock.AnyContext(), mock.Equal(&dto.GetSessionReq{
					SID: "valid-sid",
				}))).ThenReturn(&dto.GetSessionResp{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					UserID:    "user-id",
				}, nil)
				mock.WhenDouble(h.EventService.CreateEvent(mock.AnyContext(), mock.Any[*dto.CreateEventReq]())).
					ThenReturn(nil, service.ErrAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectedCookie: fmt.Sprintf("X-Session-Id=valid-sid; HttpOnly; Path=/; Max-Age=%d", mockTTL),
			expectedBody:   `{"message":"event already exists"}`,
		},
		{
			name:        "service error",
			cookie:      "X-Session-Id=valid-sid",
			requestBody: createEventRequestBody("Test Event", "", "Test Address", "2023-01-01T10:00:00Z", "2023-01-01T12:00:00Z"),
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.SessionService.GetSession(mock.AnyContext(), mock.Equal(&dto.GetSessionReq{
					SID: "valid-sid",
				}))).ThenReturn(&dto.GetSessionResp{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					UserID:    "user-id",
				}, nil)
				mock.WhenDouble(h.EventService.CreateEvent(mock.AnyContext(), mock.Any[*dto.CreateEventReq]())).
					ThenReturn(nil, errors.New("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"internal error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := mock.NewMockController(t, mockopts.StrictVerify())
			sessionService := mock.Mock[router.SessionService](ctrl)
			userService := mock.Mock[router.UserService](ctrl)
			eventService := mock.Mock[router.EventService](ctrl)
			h := newHandler(t, sessionService, userService, eventService)
			if tt.setup != nil {
				tt.setup(h)
			}

			req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			w := httptest.NewRecorder()

			srv, err := oas.NewServer(h)
			require.NoError(t, err)

			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			require.Equal(t, tt.expectedCookie, resp.Header.Get("Set-Cookie"))
			body, err := io.ReadAll(resp.Body)
			if tt.expectedBody != "" {
				require.NoError(t, err)
				require.JSONEq(t, tt.expectedBody, string(body))
			} else {
				require.Empty(t, body)
			}
		})
	}
}

func TestHandler_APIGetEvents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cookie         string
		queryParams    map[string]string
		setup          func(h *router.Handler)
		expectedStatus int
		expectedCookie string
		expectedBody   string
	}{
		{
			name: "successful get events",
			queryParams: map[string]string{
				"limit":  "10",
				"offset": "0",
			},
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvents(mock.AnyContext(), mock.Equal(&dto.GetEventsReq{
					Title:  "",
					Limit:  10,
					Offset: 0,
				}))).ThenReturn(&dto.GetEventsResp{
					Events: []dto.EventData{
						{
							ID:          "event-1",
							Title:       "Event 1",
							Description: "Desc 1",
							Location: dto.EventLocation{
								Address: "Address 1",
							},
							CreatedAt:  time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
							CreatedBy:  "user-1",
							StartedAt:  time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
							FinishedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
						},
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody: `{
				"events": [
					{
						"id": "event-1",
						"title": "Event 1",
						"description": "Desc 1",
						"location": {"address": "Address 1"},
						"created_at": "2023-01-01T10:00:00Z",
						"created_by": "user-1",
						"started_at": "2023-01-01T10:00:00Z",
						"finished_at": "2023-01-01T12:00:00Z"
					}
				],
				"count": 1
			}`,
		},
		{
			name: "get events with title filter",
			queryParams: map[string]string{
				"title":  "test",
				"limit":  "5",
				"offset": "0",
			},
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvents(mock.AnyContext(), mock.Equal(&dto.GetEventsReq{
					Title:  "test",
					Limit:  5,
					Offset: 0,
				}))).ThenReturn(&dto.GetEventsResp{Events: []dto.EventData{}}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"events":[],"count":0}`,
		},
		{
			name: "invalid limit",
			queryParams: map[string]string{
				"limit":  "-1",
				"offset": "0",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"message":"invalid \"limit\" parameter"}`,
		},
		{
			name: "invalid offset",
			queryParams: map[string]string{
				"limit":  "10",
				"offset": "-5",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"message":"invalid \"offset\" parameter"}`,
		},
		{
			name: "service error",
			queryParams: map[string]string{
				"limit":  "10",
				"offset": "0",
			},
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvents(mock.AnyContext(), mock.Any[*dto.GetEventsReq]())).
					ThenReturn(nil, errors.New("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCookie: "",
			expectedBody:   `{"message":"internal error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := mock.NewMockController(t, mockopts.StrictVerify())
			sessionService := mock.Mock[router.SessionService](ctrl)
			userService := mock.Mock[router.UserService](ctrl)
			eventService := mock.Mock[router.EventService](ctrl)
			h := newHandler(t, sessionService, userService, eventService)
			if tt.setup != nil {
				tt.setup(h)
			}

			req := httptest.NewRequest(http.MethodGet, "/events", http.NoBody)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()
			if tt.cookie != "" {
				req.Header.Set("Cookie", tt.cookie)
			}
			w := httptest.NewRecorder()

			srv, err := oas.NewServer(h)
			require.NoError(t, err)

			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			require.Equal(t, tt.expectedCookie, resp.Header.Get("Set-Cookie"))
			if tt.expectedBody != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.JSONEq(t, tt.expectedBody, string(body))
			}
		})
	}
}

func newHandler(
	t *testing.T,
	sessionService router.SessionService,
	userService router.UserService,
	eventService router.EventService,
) *router.Handler {
	t.Helper()

	return router.NewHandler(logger.NewWithOutput("debug", io.Discard), sessionService, userService, eventService, mockTTL)
}

func createEventRequestBody(title string, description string, address string, startedAt string, finishedAt string) string {
	return fmt.Sprintf(
		`{"title":%q,"description":%q,"address":%q,"started_at":%q,"finished_at":%q}`,
		title,
		description,
		address,
		startedAt,
		finishedAt,
	)
}
