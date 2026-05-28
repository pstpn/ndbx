package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	oas "ndbx/internal/router/ogen"
	"ndbx/internal/service"
	"ndbx/internal/service/dto"
	httpv "ndbx/pkg/httpserver"
	"ndbx/pkg/logger"
)

const (
	defaultLimit  = 10
	defaultOffset = 0
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
	GetUsers(ctx context.Context, req *dto.GetUsersReq) (*dto.GetUsersResp, error)
	GetUser(ctx context.Context, req *dto.GetUserReq) (*dto.GetUserResp, error)
}

type EventService interface {
	CreateEvent(ctx context.Context, req *dto.CreateEventReq) (*dto.CreateEventResp, error)
	GetEvents(ctx context.Context, req *dto.GetEventsReq) (*dto.GetEventsResp, error)
	GetEvent(ctx context.Context, req *dto.GetEventReq) (*dto.GetEventResp, error)
	PatchEvent(ctx context.Context, req *dto.PatchEventReq) error
	LikeEvent(ctx context.Context, req *dto.ReactEventReq) error
	DislikeEvent(ctx context.Context, req *dto.ReactEventReq) error
}

type ReviewService interface {
	CreateReview(ctx context.Context, req *dto.CreateReviewReq) (*dto.CreateReviewResp, error)
	GetReviews(ctx context.Context, req *dto.GetReviewsReq) (*dto.GetReviewsResp, error)
	UpdateReview(ctx context.Context, req *dto.UpdateReviewReq) error
}

type RecommendationService interface {
	GetRecommendations(ctx context.Context, req *dto.GetRecommendationsReq) (*dto.GetRecommendationsResp, error)
}

type Handler struct {
	l                     logger.Interface
	SessionService        SessionService
	UserService           UserService
	EventService          EventService
	ReviewService         ReviewService
	RecommendationService RecommendationService
	sessionTTLSeconds     int
}

func NewHandler(
	l logger.Interface,
	sessionService SessionService,
	userService UserService,
	eventService EventService,
	reviewService ReviewService,
	recommendationService RecommendationService,
	sessionTTLSeconds int,
) *Handler {
	return &Handler{
		l:                     l,
		SessionService:        sessionService,
		UserService:           userService,
		EventService:          eventService,
		ReviewService:         reviewService,
		RecommendationService: recommendationService,
		sessionTTLSeconds:     sessionTTLSeconds,
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
		session, err := h.SessionService.CreateSession(ctx, &dto.CreateSessionReq{UserID: ""})
		if err != nil {
			h.l.Errorf("failed to create session: %s", err.Error())
			return NewInternalError(), nil
		}

		return &oas.APISessionCreated{SetCookie: formSetCookie(session.SID, int(session.TTL.Seconds()))}, nil
	}

	session, err := h.SessionService.CreateOrExtendSession(ctx, &dto.CreateOrExtendSessionReq{SID: sid, UserID: ""})
	if err != nil {
		h.l.Errorf("failed to create or extend session: %s", err.Error())
		return NewInternalError(), nil
	}
	setCookie := formSetCookie(session.SID, int(session.TTL.Seconds()))

	if session.IsCreated {
		return &oas.APISessionCreated{SetCookie: setCookie}, nil
	}
	return &oas.APISessionOK{SetCookie: setCookie}, nil
}

func (h *Handler) APIRegister(ctx context.Context, req *oas.UserRegisterRequest, params oas.APIRegisterParams) (oas.APIRegisterRes, error) {
	sid := extractSID(params.Cookie.Value)
	setCookie := formSetCookie(sid, h.sessionTTLSeconds)

	if err := httpv.NotEmpty("full_name", req.FullName); err != nil {
		return NewBadRequestError(setCookie, err), nil
	}
	if err := httpv.NotEmpty("username", req.Username); err != nil {
		return NewBadRequestError(setCookie, err), nil
	}
	if err := httpv.NotEmpty("password", req.Password); err != nil {
		return NewBadRequestError(setCookie, err), nil
	}

	resp, err := h.UserService.Register(ctx, &dto.RegisterReq{FullName: req.FullName, Username: req.Username, Password: req.Password})
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			return NewConflictError(setCookie, ErrUserAlreadyExists), nil
		}

		h.l.Errorf("failed to register user: %s", err.Error())
		return NewInternalError(), nil
	}

	if sid == "" {
		session, err := h.SessionService.CreateSession(ctx, &dto.CreateSessionReq{UserID: resp.ID})
		if err != nil {
			h.l.Errorf("failed to create session: %s", err.Error())
			return NewInternalError(), nil
		}
		setCookie = formSetCookie(session.SID, h.sessionTTLSeconds)
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

	authResp, err := h.UserService.Authenticate(ctx, &dto.AuthenticateReq{Username: req.Username, Password: req.Password})
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return NewInvalidCredentialsError(setCookie), nil
		}

		h.l.Errorf("failed to authenticate user: %s", err.Error())
		return NewInternalError(), nil
	}

	if sid == "" {
		session, err := h.SessionService.CreateSession(ctx, &dto.CreateSessionReq{UserID: authResp.ID})
		if err != nil {
			h.l.Errorf("failed to create session: %s", err.Error())
			return NewInternalError(), nil
		}

		return &oas.APILoginNoContent{SetCookie: formSetCookie(session.SID, h.sessionTTLSeconds)}, nil
	}

	session, err := h.SessionService.CreateOrExtendSession(ctx, &dto.CreateOrExtendSessionReq{SID: sid, UserID: authResp.ID})
	if err != nil {
		h.l.Errorf("failed to extend session: %s", err.Error())
		return NewInternalError(), nil
	}

	return &oas.APILoginNoContent{SetCookie: formSetCookie(session.SID, h.sessionTTLSeconds)}, nil
}

func (h *Handler) APILogout(ctx context.Context, params oas.APILogoutParams) (oas.APILogoutRes, error) {
	sid := extractSID(params.Cookie.Value)

	if sid != "" {
		err := h.SessionService.DeleteSession(ctx, &dto.DeleteSessionReq{SID: sid})
		if err != nil {
			h.l.Errorf("failed to delete session: %s", err.Error())
			return NewInternalError(), nil
		}

		return &oas.APILogoutNoContent{SetCookie: formSetCookie(sid, 0)}, nil
	}

	return &oas.APILogoutUnauthorized{SetCookie: formSetCookie(sid, h.sessionTTLSeconds)}, nil
}

func (h *Handler) APICreateEvent(ctx context.Context, req *oas.CreateEventRequest, params oas.APICreateEventParams) (oas.APICreateEventRes, error) {
	sid := extractSID(params.Cookie.Value)
	setCookie := formSetCookie(sid, h.sessionTTLSeconds)
	if sid == "" {
		return &oas.APICreateEventUnauthorized{SetCookie: setCookie}, nil
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

	session, err := h.SessionService.GetSession(ctx, &dto.GetSessionReq{SID: sid})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &oas.APICreateEventUnauthorized{SetCookie: setCookie}, nil
		}

		h.l.Errorf("failed to get session: %s", err.Error())
		return NewInternalError(), nil
	}
	if session.UserID == "" {
		return &oas.APICreateEventUnauthorized{SetCookie: setCookie}, nil
	}

	eventResp, err := h.EventService.CreateEvent(ctx, &dto.CreateEventReq{
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

	if _, err := h.SessionService.CreateOrExtendSession(ctx, &dto.CreateOrExtendSessionReq{SID: sid, UserID: session.UserID}); err != nil {
		h.l.Errorf("failed to extend session: %s", err.Error())
	}

	return &oas.APICreateEventCreatedHeaders{
		SetCookie: setCookie,
		Response:  oas.APICreateEventCreated{ID: eventResp.ID},
	}, nil
}

//nolint:gocritic // params type is generated by ogen and intentionally value-based.
func (h *Handler) APIGetEvents(ctx context.Context, params oas.APIGetEventsParams) (oas.APIGetEventsRes, error) {
	setCookie := formSetCookie(extractSID(params.Cookie.Value), h.sessionTTLSeconds)

	limit, offset := int64(defaultLimit), int64(defaultOffset)
	if params.Limit.IsSet() {
		if err := httpv.NotNegativeField("limit", params.Limit.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		limit = params.Limit.Value
	}
	if params.Offset.IsSet() {
		if err := httpv.NotNegativeField("offset", params.Offset.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		offset = params.Offset.Value
	}

	var priceFrom *int64
	if params.PriceFrom.IsSet() {
		if err := httpv.NotNegativeField("price_from", params.PriceFrom.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		priceFrom = &params.PriceFrom.Value
	}

	var priceTo *int64
	if params.PriceTo.IsSet() {
		if err := httpv.NotNegativeField("price_to", params.PriceTo.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		priceTo = &params.PriceTo.Value
	}

	dateFrom, err := parseDateFilter("date_from", params.DateFrom)
	if err != nil {
		return NewBadRequestError(setCookie, err), nil
	}
	dateTo, err := parseDateFilter("date_to", params.DateTo)
	if err != nil {
		return NewBadRequestError(setCookie, err), nil
	}

	resp, err := h.EventService.GetEvents(ctx, &dto.GetEventsReq{
		ID:               params.ID.Value,
		Title:            params.Title.Value,
		Category:         string(params.Category.Value),
		PriceFrom:        priceFrom,
		PriceTo:          priceTo,
		Address:          params.Address.Value,
		City:             params.City.Value,
		DateFrom:         dateFrom,
		DateTo:           dateTo,
		UserID:           params.UserID.Value,
		User:             params.User.Value,
		Limit:            limit,
		Offset:           offset,
		IncludeReactions: includeReactions(params.Include.Value),
		IncludeReviews:   includeReviews(params.Include.Value),
	})
	if err != nil {
		h.l.Errorf("failed to get events: %s", err.Error())
		return NewInternalError(), nil
	}

	withReactions := includeReactions(params.Include.Value)
	withReviews := includeReviews(params.Include.Value)
	events := make([]oas.EventData, len(resp.Events))
	for i, event := range resp.Events {
		events[i] = toOASEventData(&event, withReactions, withReviews)
	}

	return &oas.GetEventsResponseHeaders{
		SetCookie: setCookie,
		Response: oas.GetEventsResponse{
			Events: events,
			Count:  int64(len(events)),
		},
	}, nil
}

func (h *Handler) APIGetEvent(ctx context.Context, params oas.APIGetEventParams) (oas.APIGetEventRes, error) {
	setCookie := formSetCookie(extractSID(params.Cookie.Value), h.sessionTTLSeconds)

	withReactions := includeReactions(params.Include.Value)
	withReviews := includeReviews(params.Include.Value)
	resp, err := h.EventService.GetEvent(ctx, &dto.GetEventReq{ID: params.ID, IncludeReactions: withReactions, IncludeReviews: withReviews})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return NewErrorResponse(http.StatusNotFound, setCookie, ErrNotFound), nil
		}
		h.l.Errorf("failed to get event: %s", err.Error())
		return NewInternalError(), nil
	}

	return &oas.EventDataHeaders{
		SetCookie: setCookie,
		Response:  toOASEventData(&resp.Event, withReactions, withReviews),
	}, nil
}

func (h *Handler) APIPatchEvent(ctx context.Context, req *oas.PatchEventRequest, params oas.APIPatchEventParams) (oas.APIPatchEventRes, error) {
	sid := extractSID(params.Cookie.Value)
	setCookie := formSetCookie(sid, h.sessionTTLSeconds)

	if _, err := h.EventService.GetEvent(ctx, &dto.GetEventReq{ID: params.ID}); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return NewErrorResponse(http.StatusNotFound, setCookie, fmt.Errorf("%w. Be sure that event exists and you are the organizer", ErrNotFound)), nil
		}
		h.l.Errorf("failed to get event before patch: %s", err.Error())
		return NewInternalError(), nil
	}

	if sid == "" {
		return &oas.APIPatchEventUnauthorized{SetCookie: setCookie}, nil
	}

	session, err := h.SessionService.GetSession(ctx, &dto.GetSessionReq{SID: sid})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &oas.APIPatchEventUnauthorized{SetCookie: setCookie}, nil
		}
		h.l.Errorf("failed to get session: %s", err.Error())
		return NewInternalError(), nil
	}
	if session.UserID == "" {
		return &oas.APIPatchEventUnauthorized{SetCookie: setCookie}, nil
	}

	var category *string
	if req.Category.IsSet() {
		v := string(req.Category.Value)
		category = &v
	}
	var city *string
	if req.City.IsSet() {
		v := req.City.Value
		city = &v
	}
	var price *int64
	if req.Price.IsSet() {
		if err := httpv.NotNegativeField("price", req.Price.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		v := req.Price.Value
		price = &v
	}

	err = h.EventService.PatchEvent(ctx, &dto.PatchEventReq{
		ID:        params.ID,
		CreatedBy: session.UserID,
		Category:  category,
		City:      city,
		Price:     price,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return NewErrorResponse(http.StatusNotFound, setCookie, fmt.Errorf("%w. Be sure that event exists and you are the organizer", ErrNotFound)), nil
		}
		h.l.Errorf("failed to patch event: %s", err.Error())
		return NewInternalError(), nil
	}

	if _, err := h.SessionService.CreateOrExtendSession(ctx, &dto.CreateOrExtendSessionReq{SID: sid, UserID: session.UserID}); err != nil {
		h.l.Errorf("failed to extend session: %s", err.Error())
	}

	return &oas.APIPatchEventNoContent{SetCookie: setCookie}, nil
}

func (h *Handler) APIGetUsers(ctx context.Context, params oas.APIGetUsersParams) (oas.APIGetUsersRes, error) {
	setCookie := formSetCookie(extractSID(params.Cookie.Value), h.sessionTTLSeconds)

	limit, offset := int64(defaultLimit), int64(defaultOffset)
	if params.Limit.IsSet() {
		if err := httpv.NotNegativeField("limit", params.Limit.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		limit = params.Limit.Value
	}
	if params.Offset.IsSet() {
		if err := httpv.NotNegativeField("offset", params.Offset.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		offset = params.Offset.Value
	}

	resp, err := h.UserService.GetUsers(ctx, &dto.GetUsersReq{ID: params.ID.Value, Name: params.Name.Value, Limit: limit, Offset: offset})
	if err != nil {
		h.l.Errorf("failed to get users: %s", err.Error())
		return NewInternalError(), nil
	}

	users := make([]oas.UserData, len(resp.Users))
	for i, user := range resp.Users {
		users[i] = oas.UserData{ID: user.ID, FullName: user.FullName, Username: user.Username}
	}

	return &oas.GetUsersResponseHeaders{SetCookie: setCookie, Response: oas.GetUsersResponse{Users: users, Count: int64(len(users))}}, nil
}

func (h *Handler) APIGetUser(ctx context.Context, params oas.APIGetUserParams) (oas.APIGetUserRes, error) {
	setCookie := formSetCookie(extractSID(params.Cookie.Value), h.sessionTTLSeconds)

	resp, err := h.UserService.GetUser(ctx, &dto.GetUserReq{ID: params.ID})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return NewErrorResponse(http.StatusNotFound, setCookie, ErrNotFound), nil
		}
		h.l.Errorf("failed to get user: %s", err.Error())
		return NewInternalError(), nil
	}

	return &oas.UserDataHeaders{SetCookie: setCookie, Response: oas.UserData{ID: resp.User.ID, FullName: resp.User.FullName, Username: resp.User.Username}}, nil
}

//nolint:gocritic // params type is generated by ogen and intentionally value-based.
func (h *Handler) APIGetUserEvents(ctx context.Context, params oas.APIGetUserEventsParams) (oas.APIGetUserEventsRes, error) {
	setCookie := formSetCookie(extractSID(params.Cookie.Value), h.sessionTTLSeconds)

	if _, err := h.UserService.GetUser(ctx, &dto.GetUserReq{ID: params.ID}); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return NewErrorResponse(http.StatusNotFound, setCookie, ErrUserNotFound), nil
		}
		h.l.Errorf("failed to get user for events: %s", err.Error())
		return NewInternalError(), nil
	}

	limit, offset := int64(defaultLimit), int64(defaultOffset)
	if params.Limit.IsSet() {
		if err := httpv.NotNegativeField("limit", params.Limit.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		limit = params.Limit.Value
	}
	if params.Offset.IsSet() {
		if err := httpv.NotNegativeField("offset", params.Offset.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		offset = params.Offset.Value
	}

	var priceFrom *int64
	if params.PriceFrom.IsSet() {
		if err := httpv.NotNegativeField("price_from", params.PriceFrom.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		priceFrom = &params.PriceFrom.Value
	}
	var priceTo *int64
	if params.PriceTo.IsSet() {
		if err := httpv.NotNegativeField("price_to", params.PriceTo.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		priceTo = &params.PriceTo.Value
	}

	dateFrom, err := parseDateFilter("date_from", params.DateFrom)
	if err != nil {
		return NewBadRequestError(setCookie, err), nil
	}
	dateTo, err := parseDateFilter("date_to", params.DateTo)
	if err != nil {
		return NewBadRequestError(setCookie, err), nil
	}

	resp, err := h.EventService.GetEvents(ctx, &dto.GetEventsReq{
		Title:            params.Title.Value,
		Category:         string(params.Category.Value),
		PriceFrom:        priceFrom,
		PriceTo:          priceTo,
		Address:          params.Address.Value,
		City:             params.City.Value,
		DateFrom:         dateFrom,
		DateTo:           dateTo,
		UserID:           params.ID,
		Limit:            limit,
		Offset:           offset,
		IncludeReactions: includeReactions(params.Include.Value),
		IncludeReviews:   includeReviews(params.Include.Value),
	})
	if err != nil {
		h.l.Errorf("failed to get user events: %s", err.Error())
		return NewInternalError(), nil
	}

	withReactions := includeReactions(params.Include.Value)
	withReviews := includeReviews(params.Include.Value)
	events := make([]oas.EventData, len(resp.Events))
	for i, event := range resp.Events {
		events[i] = toOASEventData(&event, withReactions, withReviews)
	}

	return &oas.GetEventsResponseHeaders{SetCookie: setCookie, Response: oas.GetEventsResponse{Events: events, Count: int64(len(events))}}, nil
}

func (h *Handler) APILikeEvent(ctx context.Context, params oas.APILikeEventParams) (oas.APILikeEventRes, error) {
	sessionState, internalErr := h.validateReactionSession(ctx, extractSID(params.Cookie.Value))
	if internalErr != nil {
		return internalErr, nil
	}
	if !sessionState.authorized {
		return &oas.APILikeEventUnauthorized{}, nil
	}

	if err := h.EventService.LikeEvent(ctx, &dto.ReactEventReq{ID: params.ID, UserID: sessionState.userID}); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return NewErrorResponse(http.StatusNotFound, sessionState.setCookie, errors.New("event not found")), nil
		}
		h.l.Errorf("failed to like event: %s", err.Error())
		return NewInternalError(), nil
	}

	h.extendSession(ctx, sessionState.sid, sessionState.userID, "like")

	return &oas.APILikeEventNoContent{SetCookie: sessionState.setCookie}, nil
}

func (h *Handler) APIDislikeEvent(ctx context.Context, params oas.APIDislikeEventParams) (oas.APIDislikeEventRes, error) {
	sessionState, internalErr := h.validateReactionSession(ctx, extractSID(params.Cookie.Value))
	if internalErr != nil {
		return internalErr, nil
	}
	if !sessionState.authorized {
		return &oas.APIDislikeEventUnauthorized{}, nil
	}

	if err := h.EventService.DislikeEvent(ctx, &dto.ReactEventReq{ID: params.ID, UserID: sessionState.userID}); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return NewErrorResponse(http.StatusNotFound, sessionState.setCookie, errors.New("event not found")), nil
		}
		h.l.Errorf("failed to dislike event: %s", err.Error())
		return NewInternalError(), nil
	}

	h.extendSession(ctx, sessionState.sid, sessionState.userID, "dislike")

	return &oas.APIDislikeEventNoContent{SetCookie: sessionState.setCookie}, nil
}

type reactionSessionState struct {
	sid        string
	userID     string
	setCookie  string
	authorized bool
}

func (h *Handler) validateReactionSession(ctx context.Context, sid string) (reactionSessionState, *oas.ErrorResponseStatusCodeWithHeaders) {
	setCookie := formSetCookie(sid, h.sessionTTLSeconds)
	if sid == "" {
		return reactionSessionState{sid: sid, setCookie: setCookie, authorized: false}, nil
	}

	session, err := h.SessionService.GetSession(ctx, &dto.GetSessionReq{SID: sid})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return reactionSessionState{sid: sid, setCookie: setCookie, authorized: false}, nil
		}
		h.l.Errorf("failed to get session before reaction: %s", err.Error())
		return reactionSessionState{}, NewInternalError()
	}
	if session.UserID == "" {
		return reactionSessionState{sid: sid, setCookie: setCookie, authorized: false}, nil
	}

	return reactionSessionState{sid: sid, userID: session.UserID, setCookie: setCookie, authorized: true}, nil
}

func (h *Handler) extendSession(ctx context.Context, sid string, userID string, action string) {
	if _, err := h.SessionService.CreateOrExtendSession(ctx, &dto.CreateOrExtendSessionReq{SID: sid, UserID: userID}); err != nil {
		h.l.Errorf("failed to extend session after %s: %s", action, err.Error())
	}
}

func toOASEventData(event *dto.EventData, withReactions bool, withReviews bool) oas.EventData {
	location := oas.LocationInfo{Address: event.Location.Address}
	if event.Location.City != "" {
		location.City = oas.NewOptString(event.Location.City)
	}

	data := oas.EventData{
		ID:          event.ID,
		Title:       event.Title,
		Description: oas.OptString{Value: event.Description, Set: event.Description != ""},
		Location:    location,
		CreatedAt:   event.CreatedAt.Format(time.RFC3339),
		CreatedBy:   event.CreatedBy,
		StartedAt:   event.StartedAt.Format(time.RFC3339),
		FinishedAt:  event.FinishedAt.Format(time.RFC3339),
	}
	if event.Category != "" {
		data.Category = oas.NewOptEventCategory(oas.EventCategory(event.Category))
	}
	if event.Category != "" || event.Price != 0 {
		data.Price = oas.NewOptInt64(event.Price)
	}
	if withReactions {
		data.Reactions = oas.NewOptEventReactions(oas.EventReactions{
			Likes:    event.Reactions.Likes,
			Dislikes: event.Reactions.Dislikes,
		})
	}
	if withReviews {
		data.Reviews = oas.NewOptEventReviews(oas.EventReviews{
			Count:  event.Reviews.Count,
			Rating: event.Reviews.Rating,
		})
	}

	return data
}

func includeReactions(include string) bool {
	parts := strings.Split(include, ",")
	for _, part := range parts {
		if strings.EqualFold(strings.TrimSpace(part), "reactions") {
			return true
		}
	}

	return false
}

func includeReviews(include string) bool {
	parts := strings.Split(include, ",")
	for _, part := range parts {
		if strings.EqualFold(strings.TrimSpace(part), "reviews") {
			return true
		}
	}

	return false
}

func parseDateFilter(field string, value oas.OptString) (*time.Time, error) {
	if !value.IsSet() || value.Value == "" {
		var parsed *time.Time
		return parsed, nil
	}
	t, err := time.Parse("20060102", value.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid \"%s\" field", field)
	}
	return &t, nil
}

func (h *Handler) APICreateEventReview(
	ctx context.Context, req *oas.CreateReviewRequest, params oas.APICreateEventReviewParams,
) (oas.APICreateEventReviewRes, error) {
	sid := extractSID(params.Cookie.Value)
	setCookie := formSetCookie(sid, h.sessionTTLSeconds)

	if !req.Comment.IsSet() || req.Comment.Value == "" {
		return NewBadRequestError(setCookie, errors.New("field \"comment\" is required")), nil
	}
	if !req.Rating.IsSet() {
		return NewBadRequestError(setCookie, errors.New("field \"rating\" is required")), nil
	}

	if sid == "" {
		return &oas.APICreateEventReviewUnauthorized{}, nil
	}

	session, err := h.SessionService.GetSession(ctx, &dto.GetSessionReq{SID: sid})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &oas.APICreateEventReviewUnauthorized{}, nil
		}
		h.l.Errorf("failed to get session: %s", err.Error())
		return NewInternalError(), nil
	}
	if session.UserID == "" {
		return &oas.APICreateEventReviewUnauthorized{}, nil
	}

	resp, err := h.ReviewService.CreateReview(ctx, &dto.CreateReviewReq{
		EventID: params.EventID,
		UserID:  session.UserID,
		Comment: req.Comment.Value,
		Rating:  int8(req.Rating.Value), //nolint:gosec // rating is validated by handler
	})
	if err != nil {
		if errors.Is(err, service.ErrAlreadyExists) {
			return &oas.CreateEventReviewConflictResponseHeaders{
				SetCookie: setCookie,
				Response:  oas.CreateEventReviewConflictResponse{Message: oas.NewOptString("review already exists")},
			}, nil
		}
		if errors.Is(err, service.ErrNotFound) {
			return NewErrorResponse(http.StatusNotFound, setCookie, ErrNotFound), nil
		}
		h.l.Errorf("failed to create review: %s", err.Error())
		return NewInternalError(), nil
	}

	h.extendSession(ctx, sid, session.UserID, "create review")

	return &oas.APICreateEventReviewCreatedHeaders{
		SetCookie: setCookie,
		Response:  oas.APICreateEventReviewCreated{ID: resp.ID},
	}, nil
}

func (h *Handler) APIGetEventReviews(ctx context.Context, params oas.APIGetEventReviewsParams) (oas.APIGetEventReviewsRes, error) {
	setCookie := formSetCookie(extractSID(params.Cookie.Value), h.sessionTTLSeconds)

	limit, offset := int64(defaultLimit), int64(defaultOffset)
	if params.Limit.IsSet() {
		if err := httpv.NotNegativeField("limit", params.Limit.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		limit = params.Limit.Value
	}
	if params.Offset.IsSet() {
		if err := httpv.NotNegativeField("offset", params.Offset.Value); err != nil {
			return NewBadRequestError(setCookie, err), nil
		}
		offset = params.Offset.Value
	}

	resp, err := h.ReviewService.GetReviews(ctx, &dto.GetReviewsReq{
		EventID: params.EventID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return NewErrorResponse(http.StatusNotFound, setCookie, ErrNotFound), nil
		}
		h.l.Errorf("failed to get reviews: %s", err.Error())
		return NewInternalError(), nil
	}

	reviews := make([]oas.ReviewData, len(resp.Reviews))
	for i, r := range resp.Reviews {
		reviews[i] = oas.ReviewData{
			ID:        r.ID,
			EventID:   r.EventID,
			Comment:   r.Comment,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
			CreatedBy: r.CreatedBy,
			Rating:    int32(r.Rating),
			UpdatedAt: r.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &oas.GetReviewsResponseHeaders{
		SetCookie: setCookie,
		Response: oas.GetReviewsResponse{
			Reviews: reviews,
			Count:   resp.Count,
		},
	}, nil
}

func (h *Handler) APIUpdateEventReview(
	ctx context.Context, req *oas.UpdateReviewRequest, params oas.APIUpdateEventReviewParams,
) (oas.APIUpdateEventReviewRes, error) {
	sid := extractSID(params.Cookie.Value)
	setCookie := formSetCookie(sid, h.sessionTTLSeconds)

	if sid == "" {
		return &oas.APIUpdateEventReviewUnauthorized{}, nil
	}

	session, err := h.SessionService.GetSession(ctx, &dto.GetSessionReq{SID: sid})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &oas.APIUpdateEventReviewUnauthorized{}, nil
		}
		h.l.Errorf("failed to get session: %s", err.Error())
		return NewInternalError(), nil
	}
	if session.UserID == "" {
		return &oas.APIUpdateEventReviewUnauthorized{}, nil
	}

	var rating *int8
	if req.Rating.IsSet() {
		v := int8(req.Rating.Value) //nolint:gosec // rating is validated by ogen schema (1-5)
		rating = &v
	}
	var comment *string
	if req.Comment.IsSet() {
		comment = &req.Comment.Value
	}

	err = h.ReviewService.UpdateReview(ctx, &dto.UpdateReviewReq{
		EventID:  params.EventID,
		ReviewID: params.ReviewID,
		UserID:   session.UserID,
		Rating:   rating,
		Comment:  comment,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &oas.UpdateEventReviewNotFoundResponseHeaders{
				SetCookie: setCookie,
				Response:  oas.UpdateEventReviewNotFoundResponse{Message: oas.NewOptString("review not found")},
			}, nil
		}
		if errors.Is(err, service.ErrForbidden) {
			return &oas.UpdateEventReviewNotFoundResponseHeaders{
				SetCookie: setCookie,
				Response:  oas.UpdateEventReviewNotFoundResponse{Message: oas.NewOptString("review not found")},
			}, nil
		}
		h.l.Errorf("failed to update review: %s", err.Error())
		return NewInternalError(), nil
	}

	h.extendSession(ctx, sid, session.UserID, "update review")

	return &oas.APIUpdateEventReviewNoContent{SetCookie: setCookie}, nil
}

func (h *Handler) APIGetRecommendations(ctx context.Context, params oas.APIGetRecommendationsParams) (oas.APIGetRecommendationsRes, error) {
	sid := extractSID(params.Cookie.Value)
	setCookie := formSetCookie(sid, h.sessionTTLSeconds)

	if sid == "" {
		return &oas.APIGetRecommendationsUnauthorized{}, nil
	}

	session, err := h.SessionService.GetSession(ctx, &dto.GetSessionReq{SID: sid})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &oas.APIGetRecommendationsUnauthorized{}, nil
		}
		h.l.Errorf("failed to get session: %s", err.Error())
		return NewInternalError(), nil
	}
	if session.UserID == "" {
		return &oas.APIGetRecommendationsUnauthorized{}, nil
	}

	resp, err := h.RecommendationService.GetRecommendations(ctx, &dto.GetRecommendationsReq{UserID: session.UserID})
	if err != nil {
		h.l.Errorf("failed to get recommendations: %s", err.Error())
		return NewInternalError(), nil
	}

	events := make([]oas.EventData, len(resp.Events))
	for i, event := range resp.Events {
		events[i] = toOASEventData(&event, false, false)
	}

	return &oas.GetRecommendationsResponseHeaders{
		SetCookie: setCookie,
		Response:  oas.GetRecommendationsResponse{Events: events},
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
