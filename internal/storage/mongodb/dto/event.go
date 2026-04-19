package dto

import "time"

type Location struct {
	Address string `bson:"address"`
	City    string `bson:"city,omitempty"`
}

type Event struct {
	ID          string    `bson:"_id,omitempty"`
	Title       string    `bson:"title"`
	Category    string    `bson:"category,omitempty"`
	Price       int64     `bson:"price,omitempty"`
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
	Events []Event
}

type GetEventReq struct {
	ID string
}

type GetEventsByTitleReq struct {
	Title string
}

type PatchEventReq struct {
	ID        string
	CreatedBy string
	Category  *string
	City      *string
	Price     *int64
}
