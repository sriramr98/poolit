package poolit

import "errors"

var (
	ErrInvalidConfig   = errors.New("invalid pool configuration")
	ErrInvalidResource = errors.New("invalid resource")
	ErrTimedOut        = errors.New("timed out waiting for resource")
	ErrPoolClosed      = errors.New("pool is closed")
	ErrDestroyFailed   = errors.New("failed to destroy resource")
	ErrResourcesActive = errors.New("resources maintained by the pool are still active and haven't been released yet")
)
