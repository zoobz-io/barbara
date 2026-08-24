# stores

Shared data access layer implementations.

## Purpose

Implement storage operations using `sum.Database`, `sum.Store`, or `sum.Bucket` wrappers. Stores are shared across all API surfaces — the same store can satisfy both public and admin contracts.

A single `Users` store might satisfy:
- `api/contracts.Users` (Get, Set, GetByLogin)
- `admin/contracts.Users` (Get, Set, Delete, List, Count)

The store implements all methods; each contract exposes only what that surface needs.

## Pattern

```go
// stores/users.go
package stores

import (
    "context"

    "github.com/jmoiron/sqlx"
    "github.com/zoobz-io/astql"
    "github.com/zoobz-io/sum"
    "github.com/zoobz-io/sumatra/models"
)

type Users struct {
    *sum.Database[models.User]
}

func NewUsers(db *sqlx.DB, renderer astql.Renderer) (*Users, error) {
    database, err := sum.NewDatabase[models.User](db, "users", renderer)
    if err != nil {
        return nil, err
    }
    return &Users{Database: database}, nil
}

// GetByLogin retrieves a user by their login name.
func (s *Users) GetByLogin(ctx context.Context, login string) (*models.User, error) {
    return s.Select().
        Where("login", "=", "login").
        Exec(ctx, map[string]any{"login": login})
}
```

## Aggregate Factory

```go
// stores/stores.go
package stores

import (
    "github.com/jmoiron/sqlx"
    "github.com/zoobz-io/astql"
    "github.com/zoobz-io/grub"
)

type Stores struct {
    Users        *Users
    Repositories *Repositories
    // ... all stores
}

func New(db *sqlx.DB, renderer astql.Renderer, bucket grub.BucketProvider) (*Stores, error) {
    users, err := NewUsers(db, renderer)
    if err != nil {
        return nil, err
    }
    // ... create all stores

    return &Stores{
        Users: users,
        // ...
    }, nil
}
```

## Multi-Surface Registration

The same store instance can be registered against different contracts in each binary:

```go
// cmd/app/main.go (public API)
allStores, _ := stores.New(db, renderer, bucket)
sum.Register[apicontracts.Users](k, allStores.Users)  // Minimal interface

// cmd/admin/main.go (admin API)
allStores, _ := stores.New(db, renderer, bucket)
sum.Register[admincontracts.Users](k, allStores.Users)  // Richer interface
```

## Guidelines

- One store per domain entity
- Embed `*sum.Database[Model]` for SQL stores
- Use `sum.NewStore[Model]` for key-value stores
- Use `sum.NewBucket[Model]` for object storage
- Create custom query methods for anything beyond basic CRUD
- Use the Soy query builder for type-safe queries
- Keep `stores.go` as the aggregate factory
- Stores are shared — implement all methods any surface might need
- Register stores against surface-specific contracts in each binary's main.go
