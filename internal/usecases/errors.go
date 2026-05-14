package usecase

import "errors"

var (
	ErrInternal = errors.New("internal error")
	ErrInvalidData = errors.New("invalid request data")
	ErrSystemAlreadyExists = errors.New("game system already exists")
)
