package dto

import "time"

type Reactions struct {
	Likes    int64
	Dislikes int64
}

type GetReactionsReq struct {
	TitleHash string
}

type GetReactionsResp struct {
	Reactions Reactions
}

type SetReactionsReq struct {
	TitleHash string
	Reactions Reactions
	TTL       time.Duration
}
