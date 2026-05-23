package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	usecase "github.com/itzLilix/questboard-session-service/internal/usecases"
)

var (
	ErrUnauthorized = errors.New("auth required")
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func handleErr(c fiber.Ctx, err error) error {
	status, code, msg := resolveErr(err)
	return c.Status(status).JSON(ErrorResponse{Code: code, Message: msg})
}

func resolveErr(err error) (int, string, string) {
	switch {
	case errors.Is(err, usecase.ErrNotFound):
		return fiber.StatusNotFound, "NOT_FOUND", "Resource not found"
	case errors.Is(err, usecase.ErrInvalidData):
		return fiber.StatusBadRequest, "INVALID_DATA", "Invalid request data"
	case errors.Is(err, usecase.ErrInvalidStatus):
		return fiber.StatusBadRequest, "INVALID_SESSION_STATUS", "Invalid session status"
	case errors.Is(err, usecase.ErrFileTooLarge):
		return fiber.StatusBadRequest, "FILE_TOO_LARGE", "File exceeds size limit"
	case errors.Is(err, usecase.ErrInvalidFileType):
		return fiber.StatusBadRequest, "INVALID_FILE_TYPE", "Unsupported file type"
	case errors.Is(err, usecase.ErrInvalidCursor):
		return fiber.StatusBadRequest, "INVALID_CURSOR", "Invalid pagination cursor"
	case errors.Is(err, ErrUnauthorized):
		return fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required"
	case errors.Is(err, usecase.ErrSystemAlreadyExists):
		return fiber.StatusConflict, "CONFLICT", "Game system already exists"
	case errors.Is(err, usecase.ErrConflict):
		return fiber.StatusConflict, "CONFLICT", "Resource already exists"
	case errors.Is(err, usecase.ErrSeatUnavailable):
		return fiber.StatusConflict, "SEAT_UNAVAILABLE", "Seat already taken"
	case errors.Is(err, usecase.ErrForbidden):
		return fiber.StatusForbidden, "FORBIDDEN", "Access forbidden"
	}
	return fiber.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error"
}