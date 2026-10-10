package tables

import "errors"

var (
	ErrNotFound  = errors.New("row not found")
	ErrDuplicate = errors.New("row already exists")
)
