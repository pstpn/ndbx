package dto

import "time"

type GetSessionReq struct {
	SID string
}

type GetSessionResp struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateSessionResp struct {
	SID           string
	MaxAgeSeconds int
}

type CreateOrExtendSessionReq struct {
	SID string
}

type CreateOrExtendSessionResp struct {
	SID           string
	MaxAgeSeconds int
	IsCreated     bool
}
