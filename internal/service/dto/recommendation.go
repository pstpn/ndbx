package dto

type GetRecommendationsReq struct {
	UserID string
}

type GetRecommendationsResp struct {
	Events []EventData
}
