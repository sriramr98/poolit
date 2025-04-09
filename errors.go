package poolit

import "errors"

var (
	ErrInvalidConfig   = errors.New("invalid pool configuration")
	ErrInvalidResource = errors.New("invalid resource")
	ErrTimedOut        = errors.New("timed out waiting for resource")
)
