package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/gosimple/slug"
	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
)

type gameSystemsUsecase struct {
	repo GameSystemsRepository
}

type CreateGameSystemInput struct {
    Name string
}

func NewGameSystemsUsecase(repo GameSystemsRepository) *gameSystemsUsecase {
	return &gameSystemsUsecase{repo: repo}
}

func (uc *gameSystemsUsecase) GetAll(ctx context.Context) ([]dtos.GameSystem, error) {
	systems, err := uc.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all systems: %w: %v", ErrInternal, err)
	}
	return systems, nil
}

func (uc *gameSystemsUsecase) GetCurated(ctx context.Context) ([]dtos.GameSystem, error) {
	systems, err := uc.repo.GetCurated(ctx)
	if err != nil {
		return nil, fmt.Errorf("get curated systems: %w: %v", ErrInternal, err)
	}
	return systems, nil
}

func (uc *gameSystemsUsecase) Search(ctx context.Context, query string) ([]dtos.GameSystem, error) {
	systems, err := uc.repo.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search systems: %w: %v", ErrInternal, err)
	}
	return systems, nil
}

func (uc *gameSystemsUsecase) AddUserSystem(ctx context.Context, input *CreateGameSystemInput) (*dtos.GameSystem, error) {
	if input.Name == "" {
		return nil, ErrInvalidData
	}
	slug := slug.Make(input.Name)
	
	userSystem := &infrastructure.CreateGameSystemParams{
		Name: input.Name,
		IsCurated: false,
		Slug: slug,
		BadgeColor: nil,
	}
	
	gs, err := uc.repo.AddGameSystem(ctx, userSystem)
	if err != nil {
		if errors.Is(err, infrastructure.ErrAlreadyExists) {
			return nil, ErrSystemAlreadyExists
		}
		return nil, fmt.Errorf("add user system: %w", err)
	}
	return gs, nil
}
