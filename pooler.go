package poolit

import (
	"fmt"
	"sync"
	"time"
)

// Pooler A thread-safe implementation of a resource pooling
type Pooler[T any] struct {
	resources           []T // List of free resources being managed by the pool
	config              PoolConfig[T]
	currentManagedCount int           //Total number of resources being managed by the pool
	sem                 chan struct{} //Semaphore implementation used to make sure that only config.MaxResources resources can be queried at any point in time. Blocks if all resources are used
	mu                  *sync.Mutex
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
		sem:       make(chan struct{}, config.MaxResources),
		mu:        &sync.Mutex{},
	}

	if err := p.createResources(config.MinResources); err != nil {
		return nil, err
	}

	p.startIdleTimer()

	return &p, nil
}

// Get fetches an available resource from the pool. If no resources are available, it will block until available
// As soon as currentManagedCount/2 resources get used, creates min(MaxResources-currentManagedCount, currentManagedCount/2) new resources
func (p *Pooler[T]) Get() T {
	// If resources are available in our pool, this unblocks immediately, else waits for someone else to release their resource
	<-p.sem

	p.mu.Lock()
	defer p.mu.Unlock()

	p.resetIdleTimer()

	resource := p.resources[0]
	p.resources = p.resources[1:]

	// Add more resources to the pool if we've utilised half of what is being managed
	if len(p.resources) <= p.currentManagedCount/2 && p.currentManagedCount < p.config.MaxResources {
		createCount := min(p.currentManagedCount/2, p.config.MaxResources-p.currentManagedCount)
		// Delegate creation to a goroutine so that we don't block the current thread
		go func() {
			if err := p.createResources(createCount); err != nil {
				fmt.Printf("Error creating resources: %v\n", err)
			}
		}()
	}

	return resource
}

// Release Adds a resource back to the pool once used
func (p *Pooler[T]) Release(resource T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resources = append(p.resources, resource)
	p.sem <- struct{}{} // add an empty struct back to the channel so that a blocked thread can read it and get resource
}

func (p *Pooler[T]) createResources(count int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// This case is possible since `createResources` can be called from multiple goroutines concurrently
	if p.currentManagedCount+count > p.config.MaxResources {
		count = p.config.MaxResources - p.currentManagedCount
	}

	for range count {
		resource, err := p.config.ResourceManager.Create()
		if err != nil {
			return err
		}
		p.resources = append(p.resources, resource)
		p.currentManagedCount++
		p.sem <- struct{}{} // Add a resource to the semaphore
	}

	return nil
}

func (p *Pooler[T]) startIdleTimer() {
	if p.config.IdleTimeout == 0 {
		return
	}

	time.AfterFunc(p.config.IdleTimeout, func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		initialCount := len(p.resources)
		if len(p.resources) > p.config.MinResources {

			// Destroy the excess resources
			for i := p.config.MinResources; i < initialCount; i++ {
				if err := p.config.ResourceManager.Destroy(p.resources[i]); err != nil {
					fmt.Println("Error destroying resource:", err)
				}
			}

			// Trim the resources to the minimum
			p.resources = p.resources[:p.config.MinResources]
			p.currentManagedCount = p.config.MinResources

			// Remove the excess resources from the semaphore
			for i := 0; i < initialCount-p.config.MinResources; i++ {
				<-p.sem
			}
		}

		// Restart the timer
		p.startIdleTimer()
	})
}

func (p *Pooler[T]) resetIdleTimer() {
	if p.config.IdleTimeout == 0 {
		return
	}

	// Cancel the existing timer and start a new one
	p.startIdleTimer()
}
