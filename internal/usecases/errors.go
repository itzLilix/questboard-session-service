package usecase

import (
	"errors"
	"fmt"

	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/cursor"
)

var (
	ErrInternal            = errors.New("internal error")
	ErrInvalidData         = errors.New("invalid request data")
	ErrInvalidCursor         = errors.New("invalid request cursor")
	ErrSystemAlreadyExists = errors.New("game system already exists")

	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("conflict")
	ErrSeatUnavailable = errors.New("no free seats")
	ErrInvalidStatus   = errors.New("invalid status")

	ErrFileTooLarge = errors.New("file too large")
	ErrInvalidFileType = errors.New("invalid file type")
)

func mapRepoErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, infrastructure.ErrAlreadyExists) {
		return ErrConflict
	}
	if errors.Is(err, infrastructure.ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, cursor.ErrInvalidCursor) {
		return fmt.Errorf("%s: %w: invalid cursor", op, ErrInvalidCursor)
	}
	return errors.Join(fmt.Errorf("%s: %w", op, err), ErrInternal)
}