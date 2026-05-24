package usecase

import (
	"errors"
	"fmt"
	"testing"

	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/cursor"
	"github.com/stretchr/testify/assert"
)

func TestMapRepoErr_NilReturnsNil(t *testing.T) {
	assert.NoError(t, mapRepoErr("op", nil))
}

func TestMapRepoErr_AlreadyExists_MapsToConflict(t *testing.T) {
	err := mapRepoErr("create", infrastructure.ErrAlreadyExists)
	assert.ErrorIs(t, err, ErrConflict)
}

func TestMapRepoErr_NotFound_MapsToNotFound(t *testing.T) {
	err := mapRepoErr("get", infrastructure.ErrNotFound)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMapRepoErr_InvalidCursor_MapsToInvalidCursor(t *testing.T) {
	err := mapRepoErr("list", cursor.ErrInvalidCursor)
	assert.ErrorIs(t, err, ErrInvalidCursor)
	assert.Contains(t, err.Error(), "list")
}

func TestMapRepoErr_WrappedAlreadyExists(t *testing.T) {
	wrapped := fmt.Errorf("db: %w", infrastructure.ErrAlreadyExists)
	err := mapRepoErr("create", wrapped)
	assert.ErrorIs(t, err, ErrConflict)
}

func TestMapRepoErr_WrappedNotFound(t *testing.T) {
	wrapped := fmt.Errorf("db: %w", infrastructure.ErrNotFound)
	err := mapRepoErr("get", wrapped)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMapRepoErr_WrappedInvalidCursor(t *testing.T) {
	wrapped := fmt.Errorf("decode: %w", cursor.ErrInvalidCursor)
	err := mapRepoErr("list", wrapped)
	assert.ErrorIs(t, err, ErrInvalidCursor)
}

func TestMapRepoErr_UnknownError_MapsToInternal(t *testing.T) {
	unknown := errors.New("connection refused")
	err := mapRepoErr("list", unknown)
	assert.ErrorIs(t, err, ErrInternal)
	assert.Contains(t, err.Error(), "list")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestMapRepoErr_OpNameInMessage(t *testing.T) {
	err := mapRepoErr("SessionRepository.GetByID", errors.New("timeout"))
	assert.Contains(t, err.Error(), "SessionRepository.GetByID")
}
