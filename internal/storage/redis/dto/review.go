package dto

import "time"

type Reviews struct {
	Count  int64
	Rating float64
}

type GetReviewsReq struct {
	TitleHash string
}

type GetReviewsResp struct {
	Reviews Reviews
}

type SetReviewsReq struct {
	TitleHash string
	Reviews   Reviews
	TTL       time.Duration
}
