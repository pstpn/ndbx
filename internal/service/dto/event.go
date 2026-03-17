package dto

import "time"

type EventLocation struct {
	Address string
}

type EventData struct {
	ID          string
	Title       string
	Description string
	Location    EventLocation
	CreatedAt   time.Time
	CreatedBy   string
	StartedAt   time.Time
	FinishedAt  time.Time
}

type CreateEventReq struct {
	Title       string
	Description string
	Address     string
	StartedAt   time.Time
	FinishedAt  time.Time
	CreatedBy   string
}

type CreateEventResp struct {
	ID string
}

type GetEventsReq struct {
	Title  string
	Limit  int64
	Offset int64
}

type GetEventsResp struct {
	Events []EventData
}
