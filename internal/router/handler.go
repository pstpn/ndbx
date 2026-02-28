package router

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	oas "ndbx/internal/router/ogen"
	"ndbx/internal/service/dto"
	"ndbx/pkg/logger"
)

type SessionService interface {
	GetSession(ctx context.Context, req *dto.GetSessionReq) (*dto.GetSessionResp, error)
	CreateSession(ctx context.Context) (*dto.CreateSessionResp, error)
	CreateOrExtendSession(ctx context.Context, req *dto.CreateOrExtendSessionReq) (*dto.CreateOrExtendSessionResp, error)
}

type Handler struct {
	l              logger.Interface
	sessionService SessionService
}

func NewHandler(l logger.Interface, sessionService SessionService) *Handler {
	return &Handler{
		l:              l,
		sessionService: sessionService,
	}
}

func (h *Handler) APIPing(_ context.Context) (oas.APIPingOK, error) {
	return oas.APIPingOK{Data: bytes.NewReader([]byte("pong"))}, nil
}

func (h *Handler) APISession(ctx context.Context, params oas.APISessionParams) (oas.APISessionRes, error) {
	var sid string
	if params.Cookie.IsSet() {
		sid = extractSID(params.Cookie.Value)
	}

	if sid == "" {
		session, err := h.sessionService.CreateSession(ctx)
		if err != nil {
			h.l.Errorf("failed to create session: %s", err.Error())
			return nil, err
		}

		return &oas.APISessionCreated{SetCookie: formSetCookie(session.SID, session.MaxAgeSeconds)}, nil
	}

	session, err := h.sessionService.CreateOrExtendSession(ctx, &dto.CreateOrExtendSessionReq{SID: sid})
	if err != nil {
		h.l.Errorf("failed to create or extend session: %s", err.Error())
		return nil, err
	}
	setCookie := formSetCookie(session.SID, session.MaxAgeSeconds)

	if session.IsCreated {
		return &oas.APISessionCreated{SetCookie: setCookie}, nil
	}
	return &oas.APISessionOK{SetCookie: setCookie}, nil
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
	return fmt.Sprintf("Set-Cookie: X-Session-Id=%s; HttpOnly; Path=/; Max-Age=%d", sid, maxAgeSeconds)
}
