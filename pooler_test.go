package poolit

import (
	"errors"
	"sync"
	"testing"
)

// MockResource represents a simple resource for testing
type MockResource struct {
	ID int
}

// MockResourceManager implements ResourceManager for testing
type MockResourceManager struct {
	createCounter  int
	destroyCounter int
	failCreate     bool
	mutex          sync.Mutex
}

func (m *MockResourceManager) Create() (MockResource, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.failCreate {
		return MockResource{}, errors.New("failed to create resource")
	}

	m.createCounter++
	return MockResource{ID: m.createCounter}, nil
}

func (m *MockResourceManager) Destroy(r MockResource) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.destroyCounter++
	return nil
}

func (m *MockResourceManager) GetCreateCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.createCounter
}

func (m *MockResourceManager) GetDestroyCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.destroyCounter
}

func TestPoolConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		config        PoolConfig[MockResource]
		expectedError error
	}{
		{
			name: "Valid config",
			config: PoolConfig[MockResource]{
				ResourceManager:    &MockResourceManager{},
				MaxResources:       10,
				MinResources:       5,
				IdleTimeoutSeconds: 30,
			},
			expectedError: nil,
		},
		{
			name: "Nil resource manager",
			config: PoolConfig[MockResource]{
				ResourceManager:    nil,
				MaxResources:       10,
				MinResources:       5,
				IdleTimeoutSeconds: 30,
			},
			expectedError: ErrInvalidConfig,
		},
		{
			name: "Zero max resources",
			config: PoolConfig[MockResource]{
				ResourceManager:    &MockResourceManager{},
				MaxResources:       0,
				MinResources:       5,
				IdleTimeoutSeconds: 30,
			},
			expectedError: ErrInvalidConfig,
		},
		{
			name: "Negative min resources",
			config: PoolConfig[MockResource]{
				ResourceManager:    &MockResourceManager{},
				MaxResources:       10,
				MinResources:       -1,
				IdleTimeoutSeconds: 30,
			},
			expectedError: ErrInvalidConfig,
		},
		{
			name: "Min resources exceed max resources",
			config: PoolConfig[MockResource]{
				ResourceManager:    &MockResourceManager{},
				MaxResources:       10,
				MinResources:       15,
				IdleTimeoutSeconds: 30,
			},
			expectedError: ErrInvalidConfig,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectedError == nil && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if tc.expectedError != nil && !errors.Is(err, tc.expectedError) {
				t.Errorf("Expected error %v, got %v", tc.expectedError, err)
			}
		})
	}
}

func TestNewPooler(t *testing.T) {
	tests := []struct {
		name           string
		config         PoolConfig[MockResource]
		failCreate     bool
		expectedError  bool
		expectedMinRes int
	}{
		{
			name: "Valid creation",
			config: PoolConfig[MockResource]{
				ResourceManager:    &MockResourceManager{},
				MaxResources:       10,
				MinResources:       5,
				IdleTimeoutSeconds: 30,
			},
			failCreate:     false,
			expectedError:  false,
			expectedMinRes: 5,
		},
		{
			name: "Zero minimum resources",
			config: PoolConfig[MockResource]{
				ResourceManager:    &MockResourceManager{},
				MaxResources:       10,
				MinResources:       0,
				IdleTimeoutSeconds: 30,
			},
			failCreate:     false,
			expectedError:  false,
			expectedMinRes: 0,
		},
		{
			name: "Resource creation failure",
			config: PoolConfig[MockResource]{
				ResourceManager:    &MockResourceManager{failCreate: true},
				MaxResources:       10,
				MinResources:       5,
				IdleTimeoutSeconds: 30,
			},
			failCreate:    true,
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rm := &MockResourceManager{failCreate: tc.failCreate}
			tc.config.ResourceManager = rm

			pooler, err := NewPooler(tc.config)

			if tc.expectedError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !tc.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if err != nil {
				return
			}

			// Verify initial resources were created
			createdCount := rm.GetCreateCount()
			if createdCount != tc.expectedMinRes {
				t.Errorf("Expected %d resources created, got %d", tc.expectedMinRes, createdCount)
			}

			// Check if managed count is correct
			if pooler.currentManagedCount != tc.expectedMinRes {
				t.Errorf("Expected currentManagedCount to be %d, got %d", tc.expectedMinRes, pooler.currentManagedCount)
			}

			// Check if resources length is correct
			if len(pooler.resources) != tc.expectedMinRes {
				t.Errorf("Expected resources length to be %d, got %d", tc.expectedMinRes, len(pooler.resources))
			}
		})
	}
}

func TestGetAndRelease(t *testing.T) {
	rm := &MockResourceManager{}
	config := PoolConfig[MockResource]{
		ResourceManager:    rm,
		MaxResources:       10,
		MinResources:       5,
		IdleTimeoutSeconds: 30,
	}

	pooler, err := NewPooler(config)
	if err != nil {
		t.Fatalf("Failed to create pooler: %v", err)
	}

	initialCount := rm.GetCreateCount()

	// Get a resource
	resource := pooler.Get()

	// Check no new resources were created for a single Get
	if rm.GetCreateCount() != initialCount {
		t.Errorf("Expected no new resources to be created, but got %d new resources",
			rm.GetCreateCount()-initialCount)
	}

	// Check resource pool size decreased
	if len(pooler.resources) != config.MinResources-1 {
		t.Errorf("Expected resources count to be %d, got %d",
			config.MinResources-1, len(pooler.resources))
	}

	// Release the resource
	pooler.Release(resource)

	// Check resource pool size is back to original
	if len(pooler.resources) != config.MinResources {
		t.Errorf("Expected resources count to be %d after release, got %d",
			config.MinResources, len(pooler.resources))
	}
}

func TestAutoScaling(t *testing.T) {
	rm := &MockResourceManager{}
	config := PoolConfig[MockResource]{
		ResourceManager:    rm,
		MaxResources:       10,
		MinResources:       4,
		IdleTimeoutSeconds: 30,
	}

	pooler, err := NewPooler(config)
	if err != nil {
		t.Fatalf("Failed to create pooler: %v", err)
	}

	// Get enough resources to trigger scaling
	var resources []MockResource
	for range 3 {
		resources = append(resources, pooler.Get())
	}

	// Wait for auto-scaling to happen
	time.Sleep(100 * time.Millisecond)

	// Check if more resources were created (autoscaling should have triggered)
	if rm.GetCreateCount() <= config.MinResources {
		t.Errorf("Expected more resources to be created due to auto-scaling, but got only %d",
			rm.GetCreateCount())
	}

	// Release all resources
	for _, r := range resources {
		pooler.Release(r)
	}
}

func TestConcurrentAccess(t *testing.T) {
	rm := &MockResourceManager{}
	config := PoolConfig[MockResource]{
		ResourceManager:    rm,
		MaxResources:       20,
		MinResources:       5,
		IdleTimeoutSeconds: 30,
	}

	pooler, err := NewPooler(config)
	if err != nil {
		t.Fatalf("Failed to create pooler: %v", err)
	}

	const numGoroutines = 15
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()

			resource := pooler.Get()
			time.Sleep(20 * time.Millisecond) // Simulate work
			pooler.Release(resource)
		}()
	}

	wg.Wait()

	// Give time for all releases to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify all resources were released back to the pool
	expectedResources := min(config.MaxResources, rm.GetCreateCount())
	if len(pooler.resources) != expectedResources {
		t.Errorf("Expected %d resources in the pool after concurrent use, got %d",
			expectedResources, len(pooler.resources))
	}
}

func TestResourceExhaustion(t *testing.T) {
	rm := &MockResourceManager{}
	config := PoolConfig[MockResource]{
		ResourceManager:    rm,
		MaxResources:       3, // Small pool for testing resource exhaustion
		MinResources:       2,
		IdleTimeoutSeconds: 30,
	}

	pooler, err := NewPooler(config)
	if err != nil {
		t.Fatalf("Failed to create pooler: %v", err)
	}

	// Get all available resources without releasing
	var resources []MockResource
	for range config.MaxResources {
		resources = append(resources, pooler.Get())
	}

	// Set up a channel to communicate when the blocked goroutine gets a resource
	done := make(chan bool)
	go func() {
		// This should block until resource is released
		_ = pooler.Get()
		done <- true
	}()

	// Check that Get() is blocked
	select {
	case <-done:
		t.Errorf("Expected Get() to block when resources exhausted, but it didn't")
	case <-time.After(50 * time.Millisecond):
		// This is expected - the call should block
	}

	// Release a resource
	pooler.Release(resources[0])

	// Now the blocked Get() should proceed
	select {
	case <-done:
		// This is expected
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Get() is still blocked after resource was released")
	}

	// Release remaining resources
	for i := 1; i < len(resources); i++ {
		pooler.Release(resources[i])
	}
}

// This test verifies that we can't create more than MaxResources through auto-scaling
func TestMaxResourcesLimit(t *testing.T) {
	rm := &MockResourceManager{}
	config := PoolConfig[MockResource]{
		ResourceManager:    rm,
		MaxResources:       5,
		MinResources:       2,
		IdleTimeoutSeconds: 30,
	}

	pooler, err := NewPooler(config)
	if err != nil {
		t.Fatalf("Failed to create pooler: %v", err)
	}

	// Get all resources to trigger scaling to max
	var resources []MockResource
	for range config.MaxResources {
		resources = append(resources, pooler.Get())
	}

	// Wait for potential auto-scaling
	time.Sleep(100 * time.Millisecond)

	// Check that we didn't exceed MaxResources
	if rm.GetCreateCount() > config.MaxResources {
		t.Errorf("Created %d resources, exceeding MaxResources of %d",
			rm.GetCreateCount(), config.MaxResources)
	}

	if pooler.currentManagedCount > config.MaxResources {
		t.Errorf("Managing %d resources, exceeding MaxResources of %d",
			pooler.currentManagedCount, config.MaxResources)
	}

	// Release all resources
	for _, r := range resources {
		pooler.Release(r)
	}
}
