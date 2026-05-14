package usecase

import (
	"errors"
	"fmt"

	"github.com/gosimple/slug"
	"github.com/itzLilix/questboard-session-service/internal/infrastructure"
	"github.com/itzLilix/questboard-shared/dtos"
)

type GameSystemsUsecase interface {
	GetCurated() ([]dtos.GameSystem, error)
	Search(query string) ([]dtos.GameSystem, error)
	AddUserSystem(input *CreateGameSystemInput) (*dtos.GameSystem, error)
}

type gameSystemsUsecase struct {
	repo GameSystemsRepository
}

type CreateGameSystemInput struct {
    Name string
}

func NewGameSystemsUsecase(repo GameSystemsRepository) GameSystemsUsecase {
	return &gameSystemsUsecase{repo: repo}
}

func (uc *gameSystemsUsecase) GetCurated() ([]dtos.GameSystem, error) {
	systems, err := uc.repo.GetCurated()
	if err != nil {
		return nil, fmt.Errorf("get curated systems: %w: %v", ErrInternal, err)
	}
	return systems, nil
}

func (uc *gameSystemsUsecase) Search(query string) ([]dtos.GameSystem, error) {
	systems, err := uc.repo.Search(query)
	if err != nil {
		return nil, fmt.Errorf("search systems: %w: %v", ErrInternal, err)
	}
	return systems, nil
}

func (uc *gameSystemsUsecase) AddUserSystem(input *CreateGameSystemInput) (*dtos.GameSystem, error) {
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
	
	gs, err := uc.repo.AddGameSystem(userSystem)
	if err != nil {
		if errors.Is(err, infrastructure.ErrAlreadyExists) {
			return nil, ErrSystemAlreadyExists
		}
		return nil, fmt.Errorf("add user system: %w", err)
	}
	return gs, nil
}
