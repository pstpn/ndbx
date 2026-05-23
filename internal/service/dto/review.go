package dto

import "time"

type ReviewData struct {
	ID        string
	EventID   string
	Rating    int8
	Comment   string
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
}

type CreateReviewReq struct {
	EventID string
	UserID  string
	Comment string
	Rating  int8
}

type CreateReviewResp struct {
	ID string
}

type GetReviewsReq struct {
	EventID string
	Limit   int64
	Offset  int64
}

type GetReviewsResp struct {
	Reviews []ReviewData
	Count   int64
}

type UpdateReviewReq struct {
	EventID  string
	ReviewID string
	UserID   string
	Rating   *int8
	Comment  *string
}
