package dto

import "time"

type Location struct {
	Address string `bson:"address"`
}

type Event struct {
	ID          string    `bson:"_id,omitempty"`
	Title       string    `bson:"title"`
	Description string    `bson:"description"`
	Location    Location  `bson:"location"`
	CreatedAt   time.Time `bson:"created_at"`
	CreatedBy   string    `bson:"created_by"`
	StartedAt   time.Time `bson:"started_at"`
	FinishedAt  time.Time `bson:"finished_at"`
}

type CreateEventReq struct {
	Title       string
	Description string
	Address     string
	StartedAt   time.Time
	FinishedAt  time.Time
	CreatedBy   string
}

type GetEventsReq struct {
	Title  string
	Limit  int64
	Offset int64
}

type GetEventsResp struct {
	Events []Event
}
