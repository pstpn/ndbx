package dto

import "time"

type GetSessionReq struct {
	SID string
}

type GetSessionResp struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    string
}

type CreateSessionReq struct {
	UserID string
}

type CreateSessionResp struct {
	SID string
	TTL time.Duration
}

type CreateOrExtendSessionReq struct {
	SID    string
	UserID string
}

type CreateOrExtendSessionResp struct {
	SID           string
	MaxAgeSeconds int
	IsCreated     bool
}

type SetUserIDReq struct {
	SID    string
	UserID string
}

type DeleteSessionReq struct {
	SID string
}
