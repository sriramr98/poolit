package poolit

// Pooler A thread-safe implementation of a resource pooling
type Pooler[T any] struct {
	resources           []T // List of free resources being managed by the pool
	config              PoolConfig[T]
	currentManagedCount int //Total number of resources being managed by the pool
}

// NewPooler Create a new instance of resource pooler with resources equal to config.MinResources. Returns an error if it's unable to create a resource
func NewPooler[T any](config PoolConfig[T]) (*Pooler[T], error) {

	if err := config.Validate(); err != nil {
		return nil, ErrInvalidConfig
	}

	var resources []T

	p := Pooler[T]{
		resources: resources,
		config:    config,
	}

	if err := p.createResources(config.MinResources); err != nil {
		return nil, err
	}

	return &p, nil
}

func (p *Pooler[T]) createResources(count int) error {
	for range count {
		resource, err := p.config.ResourceManager.Create()
		if err != nil {
			return err
		}
		p.resources = append(p.resources, resource)
		p.currentManagedCount++
	}

	return nil
}
