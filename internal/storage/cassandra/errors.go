package cassandra

import "errors"

var (
	ErrReviewAlreadyExists = errors.New("review already exists")
	ErrReviewNotFound      = errors.New("review not found")
)
