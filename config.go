package poolit

import (
	"fmt"
	"time"
)

type ResourceManager[T any] interface {
	Create() (T, error)
	Destroy(T) error
	Valid(T) bool
}

type PoolConfig[T any] struct {
	ResourceManager ResourceManager[T]

	MaxResources int           //MaxResources that can be created for use
	MinResources int           //Min number of resources to be maintained at any point in time
	IdleTimeout  time.Duration //Time (seconds, minutes, hours based on the duration value) the entire pool is idle before trimming the resources to the MinResources
}

// Validate checks if the configuration is valid
func (c PoolConfig[T]) Validate() error {
	if c.ResourceManager == nil {
		return fmt.Errorf("%w: resource manager cannot be nil", ErrInvalidConfig)
	}
	if c.MaxResources <= 0 {
		return fmt.Errorf("%w: max resources must be greater than 0", ErrInvalidConfig)
	}
	if c.MinResources < 0 {
		return fmt.Errorf("%w: min resources cannot be negative", ErrInvalidConfig)
	}
	if c.MinResources > c.MaxResources {
		return fmt.Errorf("%w: min resources cannot exceed max resources", ErrInvalidConfig)
	}
	return nil
}
