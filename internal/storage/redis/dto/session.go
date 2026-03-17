package dto

import "time"

type SessionValue struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    string    `json:"user_id"`
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

type SetUserIDReq struct {
	SID    string
	UserID string
}

type DeleteReq struct {
	SID string
}
