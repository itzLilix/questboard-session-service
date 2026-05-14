package usecase

import (
	"fmt"

	"github.com/itzLilix/questboard-shared/dtos"
)

type GameSystemsUsecase interface {
	GetCurated() ([]dtos.GameSystem, error)
	Search(query string) ([]dtos.GameSystem, error)
}

type gameSystemsUsecase struct {
	repo GameSystemsRepository
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
