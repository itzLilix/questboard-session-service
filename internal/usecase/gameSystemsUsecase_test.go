package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newGameSystemsUC(t *testing.T) (*gameSystemsUsecase, *MockGameSystemsRepository) {
	repo := NewMockGameSystemsRepository(t)
	uc := NewGameSystemsUsecase(repo)
	return uc, repo
}

var sampleSystems = []dtos.GameSystem{
	{Id: "1", Slug: "dnd-5e", Name: "D&D 5e", IsCurated: true},
	{Id: "2", Slug: "pathfinder", Name: "Pathfinder", IsCurated: true},
}

// ---------------------------------------------------------------------------
// GetAll
// ---------------------------------------------------------------------------

func TestGameSystems_GetAll_Success(t *testing.T) {
	uc, repo := newGameSystemsUC(t)
	repo.EXPECT().GetAll(mock.Anything).Return(sampleSystems, nil)

	result, err := uc.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGameSystems_GetAll_RepoError(t *testing.T) {
	uc, repo := newGameSystemsUC(t)
	repo.EXPECT().GetAll(mock.Anything).Return(nil, errors.New("db down"))

	_, err := uc.GetAll(context.Background())
	assert.ErrorIs(t, err, ErrInternal)
}

// ---------------------------------------------------------------------------
// GetCurated
// ---------------------------------------------------------------------------

func TestGameSystems_GetCurated_Success(t *testing.T) {
	uc, repo := newGameSystemsUC(t)
	repo.EXPECT().GetCurated(mock.Anything).Return(sampleSystems[:1], nil)

	result, err := uc.GetCurated(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestGameSystems_GetCurated_RepoError(t *testing.T) {
	uc, repo := newGameSystemsUC(t)
	repo.EXPECT().GetCurated(mock.Anything).Return(nil, errors.New("db down"))

	_, err := uc.GetCurated(context.Background())
	assert.ErrorIs(t, err, ErrInternal)
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

func TestGameSystems_Search_Success(t *testing.T) {
	uc, repo := newGameSystemsUC(t)
	repo.EXPECT().Search(mock.Anything, "dnd").Return(sampleSystems[:1], nil)

	result, err := uc.Search(context.Background(), "dnd")
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestGameSystems_Search_RepoError(t *testing.T) {
	uc, repo := newGameSystemsUC(t)
	repo.EXPECT().Search(mock.Anything, "q").Return(nil, errors.New("timeout"))

	_, err := uc.Search(context.Background(), "q")
	assert.ErrorIs(t, err, ErrInternal)
}

// ---------------------------------------------------------------------------
// AddUserSystem
// ---------------------------------------------------------------------------

func TestGameSystems_AddUserSystem_Success(t *testing.T) {
	uc, repo := newGameSystemsUC(t)
	expected := &dtos.GameSystem{Id: "3", Slug: "my-system", Name: "My System"}

	repo.EXPECT().AddGameSystem(mock.Anything, mock.MatchedBy(func(p *infrastructure.CreateGameSystemParams) bool {
		return p.Name == "My System" && p.Slug == "my-system" && !p.IsCurated
	})).Return(expected, nil)

	result, err := uc.AddUserSystem(context.Background(), &CreateGameSystemInput{Name: "My System"})
	require.NoError(t, err)
	assert.Equal(t, "3", result.Id)
	assert.Equal(t, "My System", result.Name)
}

func TestGameSystems_AddUserSystem_EmptyName(t *testing.T) {
	uc, _ := newGameSystemsUC(t)

	_, err := uc.AddUserSystem(context.Background(), &CreateGameSystemInput{Name: ""})
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestGameSystems_AddUserSystem_Duplicate(t *testing.T) {
	uc, repo := newGameSystemsUC(t)
	repo.EXPECT().AddGameSystem(mock.Anything, mock.Anything).
		Return(nil, infrastructure.ErrAlreadyExists)

	_, err := uc.AddUserSystem(context.Background(), &CreateGameSystemInput{Name: "Existing"})
	assert.ErrorIs(t, err, ErrSystemAlreadyExists)
}

func TestGameSystems_AddUserSystem_RepoError(t *testing.T) {
	uc, repo := newGameSystemsUC(t)
	repo.EXPECT().AddGameSystem(mock.Anything, mock.Anything).
		Return(nil, errors.New("disk full"))

	_, err := uc.AddUserSystem(context.Background(), &CreateGameSystemInput{Name: "Test"})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrSystemAlreadyExists)
}
