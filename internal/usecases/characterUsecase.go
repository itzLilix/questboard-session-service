package usecase

import (
	"context"

	"github.com/itzLilix/questboard-session-service/internal/entities"
	"github.com/itzLilix/questboard-shared/dtos"
)

type CharacterUsecase interface {
	ListMine(ctx context.Context, v *entities.Viewer, campaignID *string) ([]dtos.Character, error)
	GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Character, error)
	Create(ctx context.Context, v *entities.Viewer, in CreateCharacterInput) (*dtos.Character, error)
	Edit(ctx context.Context, id string, v *entities.Viewer, in EditCharacterInput) (*dtos.Character, error)
	Delete(ctx context.Context, id string, v *entities.Viewer) error
}

type characterUsecase struct{}

func NewCharacterUsecase() CharacterUsecase { return &characterUsecase{} }

type CreateCharacterInput struct {
	Name        string
	Class       *string
	Level       *int
	AvatarURL   *string
	Description *string
	SheetURL    *string
}

type EditCharacterInput struct {
	Name        *string
	Class       *string
	Level       *int
	AvatarURL   *string
	Description *string
	SheetURL    *string
}

func (uc *characterUsecase) ListMine(ctx context.Context, v *entities.Viewer, campaignID *string) ([]dtos.Character, error) {
	return nil, ErrNotFound
}

func (uc *characterUsecase) GetByID(ctx context.Context, id string, v *entities.Viewer) (*dtos.Character, error) {
	return nil, ErrNotFound
}

func (uc *characterUsecase) Create(ctx context.Context, v *entities.Viewer, in CreateCharacterInput) (*dtos.Character, error) {
	return nil, ErrNotFound
}

func (uc *characterUsecase) Edit(ctx context.Context, id string, v *entities.Viewer, in EditCharacterInput) (*dtos.Character, error) {
	return nil, ErrNotFound
}

func (uc *characterUsecase) Delete(ctx context.Context, id string, v *entities.Viewer) error {
	return ErrNotFound
}
