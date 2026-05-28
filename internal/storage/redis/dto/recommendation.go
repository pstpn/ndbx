package dto

import "time"

type RecommendationEvent struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Category    string `json:"category"`
	Price       int64  `json:"price"`
	Description string `json:"description"`
	City        string `json:"city"`
	Address     string `json:"address"`
	CreatedAt   string `json:"created_at"`
	CreatedBy   string `json:"created_by"`
	StartedAt   string `json:"started_at"`
	FinishedAt  string `json:"finished_at"`
}

type GetRecommendationReq struct {
	UserID string
}

type GetRecommendationResp struct {
	Events []RecommendationEvent
}

type SetRecommendationReq struct {
	UserID string
	Events []RecommendationEvent
	TTL    time.Duration
}
