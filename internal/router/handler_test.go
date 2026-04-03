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
			expectedBody:   "",
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
			expectedBody:   `{"message":"invalid \"limit\" field"}`,
		},
		{
			name: "invalid offset",
			queryParams: map[string]string{
				"limit":  "10",
				"offset": "-5",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"message":"invalid \"offset\" field"}`,
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

func TestHandler_APIGetEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cookie         string
		eventID        string
		setup          func(h *router.Handler)
		expectedStatus int
		expectedCookie string
		expectedBody   string
	}{
		{
			name:    "event found",
			eventID: "event-1",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvent(mock.AnyContext(), mock.Equal(&dto.GetEventReq{ID: "event-1"}))).
					ThenReturn(&dto.GetEventResp{Event: dto.EventData{
						ID:          "event-1",
						Title:       "Event 1",
						Category:    "party",
						Price:       1000,
						Description: "Desc",
						Location: dto.EventLocation{
							Address: "Address",
							City:    "Moscow",
						},
						CreatedAt:  time.Date(2026, 3, 14, 14, 59, 32, 0, time.UTC),
						CreatedBy:  "user-1",
						StartedAt:  time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
						FinishedAt: time.Date(2026, 4, 1, 23, 0, 0, 0, time.UTC),
					}}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody: `{
				"id":"event-1",
				"title":"Event 1",
				"category":"party",
				"price":1000,
				"description":"Desc",
				"location":{"address":"Address","city":"Moscow"},
				"created_at":"2026-03-14T14:59:32Z",
				"created_by":"user-1",
				"started_at":"2026-04-01T12:00:00Z",
				"finished_at":"2026-04-01T23:00:00Z"
			}`,
		},
		{
			name:    "event not found",
			eventID: "missing-id",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvent(mock.AnyContext(), mock.Equal(&dto.GetEventReq{ID: "missing-id"}))).
					ThenReturn(nil, service.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"message":"Not found"}`,
		},
		{
			name:    "event internal error",
			eventID: "event-1",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvent(mock.AnyContext(), mock.Any[*dto.GetEventReq]())).
					ThenReturn(nil, errors.New("boom"))
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

			req := httptest.NewRequest(http.MethodGet, "/events/"+tt.eventID, http.NoBody)
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

func TestHandler_APIPatchEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cookie         string
		eventID        string
		requestBody    string
		setup          func(h *router.Handler)
		expectedStatus int
		expectedCookie string
		expectedBody   string
	}{
		{
			name:        "unauthorized without session",
			eventID:     "event-1",
			requestBody: `{"category":"party","price":1000,"city":"Moscow"}`,
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvent(mock.AnyContext(), mock.Equal(&dto.GetEventReq{ID: "event-1"}))).
					ThenReturn(&dto.GetEventResp{Event: dto.EventData{ID: "event-1"}}, nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedCookie: "X-Session-Id=; HttpOnly; Path=/; Max-Age=0",
			expectedBody:   `{"message":"Not found. Be sure that event exists and you are the organizer"}`,
		},
		{
			name:        "unauthorized unknown session",
			cookie:      "X-Session-Id=bad-sid",
			eventID:     "event-1",
			requestBody: `{"category":"party","price":1000,"city":"Moscow"}`,
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvent(mock.AnyContext(), mock.Equal(&dto.GetEventReq{ID: "event-1"}))).
					ThenReturn(&dto.GetEventResp{Event: dto.EventData{ID: "event-1"}}, nil)
				mock.WhenDouble(h.SessionService.GetSession(mock.AnyContext(), mock.Equal(&dto.GetSessionReq{SID: "bad-sid"}))).
					ThenReturn(nil, service.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedCookie: "X-Session-Id=bad-sid; HttpOnly; Path=/; Max-Age=10",
			expectedBody:   `{"message":"Not found. Be sure that event exists and you are the organizer"}`,
		},
		{
			name:        "not found for non-existent event",
			cookie:      "X-Session-Id=sid-1",
			eventID:     "missing-id",
			requestBody: `{"category":"party","price":1000,"city":"Moscow"}`,
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvent(mock.AnyContext(), mock.Equal(&dto.GetEventReq{ID: "missing-id"}))).
					ThenReturn(nil, service.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedCookie: "X-Session-Id=sid-1; HttpOnly; Path=/; Max-Age=10",
			expectedBody:   `{"message":"Not found. Be sure that event exists and you are the organizer"}`,
		},
		{
			name:        "forbidden for чужое событие mapped to not found",
			cookie:      "X-Session-Id=sid-1",
			eventID:     "event-1",
			requestBody: `{"category":"party","price":1000,"city":"Moscow"}`,
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvent(mock.AnyContext(), mock.Any[*dto.GetEventReq]())).
					ThenReturn(&dto.GetEventResp{Event: dto.EventData{ID: "event-1"}}, nil)
				mock.WhenDouble(h.SessionService.GetSession(mock.AnyContext(), mock.Equal(&dto.GetSessionReq{SID: "sid-1"}))).
					ThenReturn(&dto.GetSessionResp{UserID: "user-1"}, nil)
				mock.WhenSingle(h.EventService.PatchEvent(mock.AnyContext(), mock.Any[*dto.PatchEventReq]())).
					ThenReturn(service.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedCookie: "X-Session-Id=sid-1; HttpOnly; Path=/; Max-Age=10",
			expectedBody:   `{"message":"Not found. Be sure that event exists and you are the organizer"}`,
		},
		{
			name:        "patch ok",
			cookie:      "X-Session-Id=sid-1",
			eventID:     "event-1",
			requestBody: `{"category":"party","price":1000,"city":"Moscow","ignored":"x"}`,
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvent(mock.AnyContext(), mock.Equal(&dto.GetEventReq{ID: "event-1"}))).
					ThenReturn(&dto.GetEventResp{Event: dto.EventData{ID: "event-1"}}, nil)
				mock.WhenDouble(h.SessionService.GetSession(mock.AnyContext(), mock.Equal(&dto.GetSessionReq{SID: "sid-1"}))).
					ThenReturn(&dto.GetSessionResp{UserID: "user-1"}, nil)
				mock.WhenSingle(h.EventService.PatchEvent(mock.AnyContext(), mock.Equal(&dto.PatchEventReq{
					ID:        "event-1",
					CreatedBy: "user-1",
					Category:  ref("party"),
					City:      ref("Moscow"),
					Price:     refI64(1000),
				}))).ThenReturn(nil)
				mock.WhenDouble(h.SessionService.CreateOrExtendSession(mock.AnyContext(), mock.Equal(&dto.CreateOrExtendSessionReq{
					SID:    "sid-1",
					UserID: "user-1",
				}))).ThenReturn(&dto.CreateOrExtendSessionResp{SID: "sid-1", TTL: time.Second * mockTTL, IsCreated: false}, nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedCookie: "X-Session-Id=sid-1; HttpOnly; Path=/; Max-Age=10",
		},
		{
			name:        "patch with empty city",
			cookie:      "X-Session-Id=sid-1",
			eventID:     "event-1",
			requestBody: `{"city":""}`,
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvent(mock.AnyContext(), mock.Equal(&dto.GetEventReq{ID: "event-1"}))).
					ThenReturn(&dto.GetEventResp{Event: dto.EventData{ID: "event-1"}}, nil)
				mock.WhenDouble(h.SessionService.GetSession(mock.AnyContext(), mock.Equal(&dto.GetSessionReq{SID: "sid-1"}))).
					ThenReturn(&dto.GetSessionResp{UserID: "user-1"}, nil)
				mock.WhenSingle(h.EventService.PatchEvent(mock.AnyContext(), mock.Equal(&dto.PatchEventReq{
					ID:        "event-1",
					CreatedBy: "user-1",
					City:      ref(""),
				}))).ThenReturn(nil)
				mock.WhenDouble(h.SessionService.CreateOrExtendSession(mock.AnyContext(), mock.Any[*dto.CreateOrExtendSessionReq]())).
					ThenReturn(&dto.CreateOrExtendSessionResp{SID: "sid-1", TTL: time.Second * mockTTL, IsCreated: false}, nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedCookie: "X-Session-Id=sid-1; HttpOnly; Path=/; Max-Age=10",
		},
		{
			name:        "patch invalid negative price",
			cookie:      "X-Session-Id=sid-1",
			eventID:     "event-1",
			requestBody: `{"price":-1}`,
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.EventService.GetEvent(mock.AnyContext(), mock.Equal(&dto.GetEventReq{ID: "event-1"}))).
					ThenReturn(&dto.GetEventResp{Event: dto.EventData{ID: "event-1"}}, nil)
				mock.WhenDouble(h.SessionService.GetSession(mock.AnyContext(), mock.Equal(&dto.GetSessionReq{SID: "sid-1"}))).
					ThenReturn(&dto.GetSessionResp{UserID: "user-1"}, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedCookie: "X-Session-Id=sid-1; HttpOnly; Path=/; Max-Age=10",
			expectedBody:   `{"message":"invalid \"price\" field"}`,
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

			req := httptest.NewRequest(http.MethodPatch, "/events/"+tt.eventID, strings.NewReader(tt.requestBody))
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

func TestHandler_APIGetUsers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		query          string
		setup          func(h *router.Handler)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:  "users found",
			query: "?name=John&limit=2&offset=1",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.GetUsers(mock.AnyContext(), mock.Equal(&dto.GetUsersReq{ID: "", Name: "John", Limit: 2, Offset: 1}))).
					ThenReturn(&dto.GetUsersResp{Users: []dto.UserData{{ID: "u1", FullName: "John Doe", Username: "john_doe"}}}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"users":[{"id":"u1","full_name":"John Doe","username":"john_doe"}],"count":1}`,
		},
		{
			name:           "invalid limit",
			query:          "?limit=-1",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"invalid \"limit\" field"}`,
		},
		{
			name:  "service error",
			query: "",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.GetUsers(mock.AnyContext(), mock.Any[*dto.GetUsersReq]())).
					ThenReturn(nil, errors.New("boom"))
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

			req := httptest.NewRequest(http.MethodGet, "/users"+tt.query, http.NoBody)
			w := httptest.NewRecorder()
			srv, err := oas.NewServer(h)
			require.NoError(t, err)
			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			if tt.expectedBody != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.JSONEq(t, tt.expectedBody, string(body))
			}
		})
	}
}

func TestHandler_APIGetUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		id             string
		setup          func(h *router.Handler)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "user found",
			id:   "u1",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.GetUser(mock.AnyContext(), mock.Equal(&dto.GetUserReq{ID: "u1"}))).
					ThenReturn(&dto.GetUserResp{User: dto.UserData{ID: "u1", FullName: "John Doe", Username: "john_doe"}}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"u1","full_name":"John Doe","username":"john_doe"}`,
		},
		{
			name: "user not found",
			id:   "missing",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.GetUser(mock.AnyContext(), mock.Equal(&dto.GetUserReq{ID: "missing"}))).
					ThenReturn(nil, service.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"message":"Not found"}`,
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

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.id, http.NoBody)
			w := httptest.NewRecorder()
			srv, err := oas.NewServer(h)
			require.NoError(t, err)
			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.JSONEq(t, tt.expectedBody, string(body))
		})
	}
}

func TestHandler_APIGetUserEvents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		id             string
		query          string
		setup          func(h *router.Handler)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:  "user events found with filters",
			id:    "u1",
			query: "?category=party&city=Moscow&price_to=0&date_from=20260314",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.GetUser(mock.AnyContext(), mock.Equal(&dto.GetUserReq{ID: "u1"}))).
					ThenReturn(&dto.GetUserResp{User: dto.UserData{ID: "u1", FullName: "John", Username: "john"}}, nil)
				dateFrom := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)
				priceTo := int64(0)
				mock.WhenDouble(h.EventService.GetEvents(mock.AnyContext(), mock.Equal(&dto.GetEventsReq{
					Category: "party",
					City:     "Moscow",
					PriceTo:  &priceTo,
					DateFrom: &dateFrom,
					UserID:   "u1",
					Limit:    10,
					Offset:   0,
				}))).ThenReturn(&dto.GetEventsResp{Events: []dto.EventData{{
					ID:          "e1",
					Title:       "Party",
					Category:    "party",
					Price:       0,
					Description: "Free party",
					Location:    dto.EventLocation{Address: "A", City: "Moscow"},
					CreatedAt:   time.Date(2026, 3, 14, 14, 59, 32, 0, time.UTC),
					CreatedBy:   "u1",
					StartedAt:   time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
					FinishedAt:  time.Date(2026, 4, 1, 23, 0, 0, 0, time.UTC),
				}}}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: `{
				"events":[{
					"id":"e1",
					"title":"Party",
					"category":"party",
					"price":0,
					"description":"Free party",
					"location":{"address":"A","city":"Moscow"},
					"created_at":"2026-03-14T14:59:32Z",
					"created_by":"u1",
					"started_at":"2026-04-01T12:00:00Z",
					"finished_at":"2026-04-01T23:00:00Z"
				}],
				"count":1
			}`,
		},
		{
			name: "user not found",
			id:   "missing",
			setup: func(h *router.Handler) {
				mock.WhenDouble(h.UserService.GetUser(mock.AnyContext(), mock.Equal(&dto.GetUserReq{ID: "missing"}))).
					ThenReturn(nil, service.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"message":"User not found"}`,
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

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.id+"/events"+tt.query, http.NoBody)
			w := httptest.NewRecorder()
			srv, err := oas.NewServer(h)
			require.NoError(t, err)
			srv.ServeHTTP(w, req)

			resp := w.Result()
			t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
			require.Equal(t, tt.expectedStatus, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.JSONEq(t, tt.expectedBody, string(body))
		})
	}
}

func ref(v string) *string {
	return &v
}

func refI64(v int64) *int64 {
	return &v
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
