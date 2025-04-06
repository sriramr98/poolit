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
