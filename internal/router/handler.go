package router

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	oas "ndbx/internal/router/ogen"
	"ndbx/internal/service"
	"ndbx/internal/service/dto"
	httpv "ndbx/pkg/httpserver"
	"ndbx/pkg/logger"
)

type SessionService interface {
	GetSession(ctx context.Context, req *dto.GetSessionReq) (*dto.GetSessionResp, error)
	CreateSession(ctx context.Context, req *dto.CreateSessionReq) (*dto.CreateSessionResp, error)
	CreateOrExtendSession(ctx context.Context, req *dto.CreateOrExtendSessionReq) (*dto.CreateOrExtendSessionResp, error)
	DeleteSession(ctx context.Context, req *dto.DeleteSessionReq) error
}

type UserService interface {
	Register(ctx context.Context, req *dto.RegisterReq) (*dto.RegisterResp, error)
	Authenticate(ctx context.Context, req *dto.AuthenticateReq) (*dto.AuthenticateResp, error)
}

type EventService interface {
	CreateEvent(ctx context.Context, req *dto.CreateEventReq) (*dto.CreateEventResp, error)
	GetEvents(ctx context.Context, req *dto.GetEventsReq) (*dto.GetEventsResp, error)
}

type Handler struct {
	l                 logger.Interface
	sessionService    SessionService
	userService       UserService
	eventService      EventService
	sessionTTLSeconds int
}

func NewHandler(
	l logger.Interface,
	sessionService SessionService,
	userService UserService,
	eventService EventService,
	sessionTTLSeconds int,
) *Handler {
	return &Handler{
		l:                 l,
		sessionService:    sessionService,
		userService:       userService,
		eventService:      eventService,
		sessionTTLSeconds: sessionTTLSeconds,
	}
}

func (h *Handler) APIHealth(_ context.Context, params oas.APIHealthParams) (*oas.HealthResponseHeaders, error) {
	return &oas.HealthResponseHeaders{
		SetCookie: oas.OptString{
			Value: formSetCookie(extractSID(params.Cookie.Value), h.sessionTTLSeconds),
			Set:   params.Cookie.IsSet(),
		},
		Response: oas.HealthResponse{Status: "ok"},
	}, nil
}

func (h *Handler) APISession(ctx context.Context, params oas.APISessionParams) (oas.APISessionRes, error) {
	sid := extractSID(params.Cookie.Value)

	if sid == "" {
		session, err := h.sessionService.CreateSession(ctx, &dto.CreateSessionReq{UserID: ""})
		if err != nil {
			h.l.Errorf("failed to create session: %s", err.Error())
			return NewInternalError(), nil
		}

		return &oas.APISessionCreated{SetCookie: formSetCookie(session.SID, int(session.TTL.Seconds()))}, nil
	}

	session, err := h.sessionService.CreateOrExtendSession(ctx, &dto.CreateOrExtendSessionReq{SID: sid, UserID: ""})
	if err != nil {
		h.l.Errorf("failed to create or extend session: %s", err.Error())
		return NewInternalError(), nil
	}
	setCookie := formSetCookie(session.SID, session.MaxAgeSeconds)

	if session.IsCreated {
		return &oas.APISessionCreated{SetCookie: setCookie}, nil
	}
	return &oas.APISessionOK{SetCookie: setCookie}, nil
}

func (h *Handler) APIRegister(ctx context.Context, req *oas.UserRegisterRequest, params oas.APIRegisterParams) (oas.APIRegisterRes, error) {
	setCookie := formSetCookie(extractSID(params.Cookie.Value), h.sessionTTLSeconds)

	if err := httpv.NotEmpty("full_name", req.FullName); err != nil {
		return NewBadRequestError(setCookie, err), nil
	}
	if err := httpv.NotEmpty("username", req.Username); err != nil {
		return NewBadRequestError(setCookie, err), nil
	}
	if err := httpv.NotEmpty("password", req.Password); err != nil {
		return NewBadRequestError(setCookie, err), nil
	}

	_, err := h.userService.Register(ctx, &dto.RegisterReq{FullName: req.FullName, Username: req.Username, Password: req.Password})
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			return NewConflictError(setCookie, ErrUserAlreadyExists), nil
		}

		h.l.Errorf("failed to register user: %s", err.Error())
		return NewInternalError(), nil
	}

	return &oas.APIRegisterCreated{SetCookie: setCookie}, nil
}

func (h *Handler) APILogin(ctx context.Context, req *oas.LoginRequest, params oas.APILoginParams) (oas.APILoginRes, error) {
	sid := extractSID(params.Cookie.Value)
	setCookie := formSetCookie(sid, h.sessionTTLSeconds)

	if httpv.NotEmpty("username", req.Username) != nil {
		return NewInvalidCredentialsError(setCookie), nil //nolint:nilerr // we don`t need this err
	}
	if httpv.NotEmpty("password", req.Password) != nil {
		return NewInvalidCredentialsError(setCookie), nil //nolint:nilerr // we don`t need this err
	}

	authResp, err := h.userService.Authenticate(ctx, &dto.AuthenticateReq{Username: req.Username, Password: req.Password})
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return NewInvalidCredentialsError(setCookie), nil
		}

		h.l.Errorf("failed to authenticate user: %s", err.Error())
		return NewInternalError(), nil
	}

	if sid == "" {
		session, err := h.sessionService.CreateSession(ctx, &dto.CreateSessionReq{UserID: authResp.ID})
		if err != nil {
			h.l.Errorf("failed to create session: %s", err.Error())
			return NewInternalError(), nil
		}

		return &oas.APILoginNoContent{SetCookie: formSetCookie(session.SID, h.sessionTTLSeconds)}, nil
	}

	session, err := h.sessionService.CreateOrExtendSession(ctx, &dto.CreateOrExtendSessionReq{SID: sid, UserID: authResp.ID})
	if err != nil {
		h.l.Errorf("failed to extend session: %s", err.Error())
		return NewInternalError(), nil
	}

	return &oas.APILoginNoContent{SetCookie: formSetCookie(session.SID, h.sessionTTLSeconds)}, nil
}

func (h *Handler) APILogout(ctx context.Context, params oas.APILogoutParams) (oas.APILogoutRes, error) {
	sid := extractSID(params.Cookie.Value)

	if sid != "" {
		err := h.sessionService.DeleteSession(ctx, &dto.DeleteSessionReq{SID: sid})
		if err != nil {
			h.l.Errorf("failed to delete session: %s", err.Error())
			return NewInternalError(), nil
		}
	}

	return &oas.APILogoutNoContent{SetCookie: formSetCookie(sid, 0)}, nil
}

func (h *Handler) APICreateEvent(ctx context.Context, req *oas.CreateEventRequest, params oas.APICreateEventParams) (oas.APICreateEventRes, error) {
	sid := extractSID(params.Cookie.Value)
	setCookie := formSetCookie(sid, h.sessionTTLSeconds)
	if sid == "" {
		return NewUnauthorizedError(setCookie, nil), nil
	}

	if err := httpv.NotEmpty("title", req.Title); err != nil {
		return NewBadRequestError(setCookie, err), nil
	}
	if err := httpv.NotEmpty("address", req.Address); err != nil {
		return NewBadRequestError(setCookie, err), nil
	}
	startedAt, err := httpv.ParseRFC3339("started_at", req.StartedAt)
	if err != nil {
		return NewBadRequestError(setCookie, err), nil
	}
	finishedAt, err := httpv.ParseRFC3339("finished_at", req.FinishedAt)
	if err != nil {
		return NewBadRequestError(setCookie, err), nil
	}

	session, err := h.sessionService.GetSession(ctx, &dto.GetSessionReq{SID: sid})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return NewUnauthorizedError(setCookie, nil), nil
		}

		h.l.Errorf("failed to get session: %s", err.Error())
		return NewInternalError(), nil
	}
	if session.UserID == "" {
		return NewUnauthorizedError(setCookie, nil), nil
	}

	eventResp, err := h.eventService.CreateEvent(ctx, &dto.CreateEventReq{
		Title:       req.Title,
		Description: req.Description.Or(""),
		Address:     req.Address,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		CreatedBy:   session.UserID,
	})
	if err != nil {
		if errors.Is(err, service.ErrAlreadyExists) {
			return NewConflictError(setCookie, ErrEventAlreadyExists), nil
		}

		h.l.Errorf("failed to create event: %s", err.Error())
		return NewInternalError(), nil
	}

	if _, err := h.sessionService.CreateOrExtendSession(ctx, &dto.CreateOrExtendSessionReq{SID: sid, UserID: session.UserID}); err != nil {
		h.l.Errorf("failed to extend session: %s", err.Error())
	}

	return &oas.APICreateEventCreatedHeaders{
		SetCookie: setCookie,
		Response:  oas.APICreateEventCreated{ID: eventResp.ID},
	}, nil
}

func (h *Handler) APIGetEvents(ctx context.Context, params oas.APIGetEventsParams) (oas.APIGetEventsRes, error) {
	setCookie := formSetCookie(extractSID(params.Cookie.Value), h.sessionTTLSeconds)

	if err := httpv.NotNegative(params.Limit, params.Offset); err != nil {
		return NewBadRequestError(setCookie, err), nil
	}
	title := params.Title.Value

	resp, err := h.eventService.GetEvents(ctx, &dto.GetEventsReq{Title: title, Limit: params.Limit, Offset: params.Offset})
	if err != nil {
		h.l.Errorf("failed to get events: %s", err.Error())
		return NewInternalError(), nil
	}

	events := make([]oas.EventData, len(resp.Events))
	for i, event := range resp.Events {
		events[i] = oas.EventData{
			ID:          event.ID,
			Title:       event.Title,
			Description: oas.OptString{Value: event.Description, Set: event.Description != ""},
			Location:    oas.LocationInfo{Address: event.Location.Address},
			CreatedAt:   event.CreatedAt.Format(time.RFC3339),
			CreatedBy:   event.CreatedBy,
			StartedAt:   event.StartedAt.Format(time.RFC3339),
			FinishedAt:  event.FinishedAt.Format(time.RFC3339),
		}
	}

	return &oas.GetEventsResponseHeaders{
		SetCookie: setCookie,
		Response: oas.GetEventsResponse{
			Events: events,
			Count:  int64(len(events)),
		},
	}, nil
}

// extractSID parses Cookie header to find X-Session-Id cookie value
func extractSID(cookieHeader string) string {
	cookies := strings.Split(cookieHeader, ";")
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		if strings.HasPrefix(cookie, "X-Session-Id=") {
			return strings.SplitN(cookie, "=", 2)[1] //nolint:mnd // always has 2 components (1: key, 2: value)
		}
	}
	return ""
}

// formSetCookie forms Set-Cookie header
func formSetCookie(sid string, maxAgeSeconds int) string {
	if sid == "" {
		maxAgeSeconds = 0
	}
	return fmt.Sprintf("X-Session-Id=%s; HttpOnly; Path=/; Max-Age=%d", sid, maxAgeSeconds)
}
