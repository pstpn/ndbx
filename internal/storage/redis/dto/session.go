package dto

import "time"

type SessionValue struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SetReq struct {
	SID   string
	Value SessionValue
	TTL   time.Duration
}

type SetOrUpdateReq struct {
	SID      string
	NewSID   string
	NewValue SessionValue
	TTL      time.Duration
}

type SetOrUpdateResp struct {
	SID       string
	IsCreated bool
}

type GetReq struct {
	SID string
}

type GetResp struct {
	SessionValue
}
