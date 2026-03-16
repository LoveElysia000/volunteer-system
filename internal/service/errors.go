package service

import "volunteer-system/internal/response"

func errBadRequest(message string) error {
	return response.NewError(response.ErrCodeBadRequest, message)
}

func errUnauthorized(message string) error {
	return response.NewError(response.ErrCodeUnauthorized, message)
}

func errForbidden(message string) error {
	return response.NewError(response.ErrCodeForbidden, message)
}

func errNotFound(message string) error {
	return response.NewError(response.ErrCodeNotFound, message)
}

