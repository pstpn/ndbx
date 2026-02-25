package router

import (
	"bytes"
	"context"

	oas "ndbx/internal/router/ogen"
	"ndbx/pkg/logger"
)

type Handler struct{ l logger.Interface }

func NewHandler(l logger.Interface) *Handler { return &Handler{l: l} }

func (h *Handler) APIPing(_ context.Context) (oas.APIPingOK, error) {
	return oas.APIPingOK{Data: bytes.NewReader([]byte("pong"))}, nil
}
