package dto

import "time"

type EventLocation struct {
	Address string
	City    string
}

type EventData struct {
	ID          string
	Title       string
	Category    string
	Price       int64
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
	ID        string
	Title     string
	Category  string
	PriceFrom *int64
	PriceTo   *int64
	Address   string
	City      string
	DateFrom  *time.Time
	DateTo    *time.Time
	UserID    string
	User      string
	Limit     int64
	Offset    int64
}

type GetEventsResp struct {
	Events []EventData
}

type GetEventReq struct {
	ID string
}

type GetEventResp struct {
	Event EventData
}

type PatchEventReq struct {
	ID        string
	CreatedBy string
	Category  *string
	City      *string
	Price     *int64
}
