package router

import (
	"errors"
	"net/http"

	api "ndbx/internal/router/ogen"
)

var (
	ErrInternal           = errors.New("internal error")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEventAlreadyExists = errors.New("event already exists")
)

func NewErrorResponse(statusCode int, setCookie string, err error) *api.ErrorResponseStatusCodeWithHeaders {
	errResp := &api.ErrorResponseStatusCodeWithHeaders{
		StatusCode: statusCode,
		SetCookie:  setCookie,
		Response:   api.ErrorResponse{},
	}
	if err != nil {
		errResp.Response.SetMessage(api.NewOptString(err.Error()))
	}

	return errResp
}

func NewInternalError() *api.ErrorResponseStatusCodeWithHeaders {
	return NewErrorResponse(http.StatusInternalServerError, "", ErrInternal)
}

func NewBadRequestError(setCookie string, err error) *api.ErrorResponseStatusCodeWithHeaders {
	return NewErrorResponse(http.StatusBadRequest, setCookie, err)
}

func NewConflictError(setCookie string, err error) *api.ErrorResponseStatusCodeWithHeaders {
	return NewErrorResponse(http.StatusConflict, setCookie, err)
}

func NewInvalidCredentialsError(setCookie string) *api.ErrorResponseStatusCodeWithHeaders {
	return NewErrorResponse(http.StatusUnauthorized, setCookie, ErrInvalidCredentials)
}

func NewUnauthorizedError(setCookie string, err error) *api.ErrorResponseStatusCodeWithHeaders {
	return NewErrorResponse(http.StatusUnauthorized, setCookie, err)
}
