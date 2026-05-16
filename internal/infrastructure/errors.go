package infrastructure

import "errors"

var (
	ErrAlreadyExists = errors.New("item already exists")
	ErrNotFound      = errors.New("row not found")
)