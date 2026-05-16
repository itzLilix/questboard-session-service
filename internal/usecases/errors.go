package usecase

import "errors"

var (
	ErrInternal            = errors.New("internal error")
	ErrInvalidData         = errors.New("invalid request data")
	ErrSystemAlreadyExists = errors.New("game system already exists")

	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("conflict")
	ErrSeatUnavailable = errors.New("no free seats")
	ErrInvalidStatus   = errors.New("invalid status transition")
	ErrInvalidURL      = errors.New("invalid or disallowed url")
)
