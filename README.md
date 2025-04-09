# poolit

[![Go Reference](https://pkg.go.dev/badge/github.com/sriramr98/poolit.svg)](https://pkg.go.dev/github.com/sriramr98/poolit)
[![Go Report Card](https://goreportcard.com/badge/github.com/sriramr98/poolit)](https://goreportcard.com/report/github.com/sriramr98/poolit)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`poolit` is a high-performance, type-safe resource pooling library for Go. Built with generics, it provides efficient management of any resource type - from database connections to file handles, HTTP clients, and more.

## Features

- **Generic Type Support**: Manage any resource type with full type safety
- **Dynamic Pool Sizing**: Automatically scales resources based on demand
- **Intelligent Scaling**: Proactively manages resources to maintain optimal performance
- **Idle Timeout**: Automatically downsizes pool during periods of inactivity
- **Context Support**: Respects context deadlines and cancellation
- **Resource Validation**: Validates resources before returning them to the pool
- **Comprehensive Stats**: Monitor pool utilization and performance
- **Thread-Safe**: Fully concurrent-safe implementation
- **Clean Shutdown**: Graceful resource cleanup on pool closure

## Installation

```bash
go get github.com/sriramr98/poolit
```

## Quick Start

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sriramr98/poolit"
	_ "github.com/lib/pq"
)

// Define your resource manager
type DBManager struct {
	connStr string
}

func (m *DBManager) Create() (*sql.DB, error) {
	return sql.Open("postgres", m.connStr)
}

func (m *DBManager) Destroy(db *sql.DB) error {
	return db.Close()
}

func main() {
	// Configure your pool
	config := poolit.PoolConfig[*sql.DB]{
		ResourceManager:    &DBManager{connStr: "postgres://user:pass@localhost/db"},
		MaxResources:       10,
		MinResources:       2,
		IdleTimeout: time.Second * 60,
	}

	// Create the pool
	pool, err := poolit.NewPooler(config)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	// Get a resource with context
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	
	db, err := pool.Get(ctx)
	if err != nil {
		panic(err)
	}

	// Use the resource
	// ...

	// Return the resource to the pool
	if err := pool.Release(db); err != nil {
		fmt.Printf("Error releasing resource: %v\n", err)
	}

	// Print pool stats
	fmt.Printf("Pool stats: %+v\n", pool.Stats())
}
```

## How It Works

### Resource Management

`poolit` uses a combination of channels and mutexes to efficiently manage resource allocation:

- **Semaphore Pattern**: Uses a buffered channel as a semaphore to control concurrent resource usage
- **Lazy Allocation**: Creates resources on-demand up to the configured maximum
- **Resource Scaling**: Proactively scales the resource pool when usage exceeds 50% of capacity
- **Resource Validation**: Validates resources before returning them to the pool

### Dynamic Scaling

The pool intelligently manages resources:

1. **Initialization**: Creates `MinResources` during pool creation
2. **Scaling Up**: When resource usage exceeds 50% of the current capacity, it proactively creates more resources
3. **Scaling Down**: During periods of inactivity, it trims excess resources down to `MinResources`

### Context Integration

Pool operations respect Go's context package for timeouts and cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
defer cancel()

// Get will respect the context deadline or cancellation
resource, err := pool.Get(ctx)
```

### Error Handling

`poolit` provides detailed error types:

- `ErrPoolClosed`: Returned when attempting to use a closed pool
- `ErrPoolTimeout`: Returned when resource acquisition times out
- `ErrInvalidConfig`: Returned for invalid pool configuration
- `ErrNoResourcesLeft`: Returned when no resources can be acquired

## Advanced Configuration

### Resource Validation

Implement the `Validate` method to check resource health before reuse:

```go
func (m *MyResourceManager) Validate(resource MyResource) bool {
    // Check if the resource is still valid and healthy
    return resource.IsHealthy()
}
```

### Custom Resource Types

Any resource type can be managed, as long as you implement the `ResourceManager` interface:

```go
type ResourceManager[T any] interface {
    Create() (T, error)
    Validate(T) bool
    Destroy(T) error
}
```

## Performance Tuning

For optimal performance, consider these tips:

1. **Min Resources**: Set this based on your typical minimum concurrent usage
2. **Max Resources**: Set this based on your resource constraints (e.g., database connection limits)
3. **Idle Timeout**: Set higher for stable workloads, lower for variable workloads

## Monitoring

Monitor pool performance using the `Stats()` method:

```go
stats := pool.Stats()
fmt.Printf("Available: %d, In use: %d, Total: %d\n", 
    stats["available_resources"].(int),
    stats["in_use_resources"].(int),
    stats["current_managed_count"].(int))
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.****