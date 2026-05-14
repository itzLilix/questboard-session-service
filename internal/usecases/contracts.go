package usecase

import "github.com/itzLilix/questboard-shared/dtos"

type GameSystemsRepository interface {
	GetCurated() ([]dtos.GameSystem, error)
	Search(q string) ([]dtos.GameSystem, error)
}