package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	usecase "github.com/itzLilix/questboard-session-service/internal/usecases"
)

func statusFor(err error) int {
	switch {
	case errors.Is(err, usecase.ErrNotFound):
		return fiber.StatusNotFound
	case errors.Is(err, usecase.ErrForbidden):
		return fiber.StatusForbidden
	case errors.Is(err, usecase.ErrConflict),
		errors.Is(err, usecase.ErrSeatUnavailable):
		return fiber.StatusConflict
	case errors.Is(err, usecase.ErrInvalidData),
		errors.Is(err, usecase.ErrInvalidStatus),
		errors.Is(err, usecase.ErrInvalidURL):
		return fiber.StatusBadRequest
	default:
		return fiber.StatusInternalServerError
	}
}
