package poolit

import "errors"

var (
	ErrInvalidConfig   = errors.New("invalid pool configuration")
	ErrInvalidResource = errors.New("invalid resource")
)
