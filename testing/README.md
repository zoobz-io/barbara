# testing

Test infrastructure: fixtures, mocks, and helpers.

## Purpose

Provide reusable test utilities that make writing tests fast and consistent. All files use the `//go:build testing` build tag.

## Fixtures

Create test data with sensible defaults:

```go
//go:build testing

package testing

import (
    "testing"

    "github.com/zoobz-io/sumatra/models"
)

func NewUser(t *testing.T) *models.User {
    t.Helper()
    return &models.User{
        ID:          1000,
        Login:       "testuser",
        Email:       "test@example.com",
        AccessToken: "test-token",
    }
}

func NewUsers(t *testing.T, n int) []*models.User {
    t.Helper()
    users := make([]*models.User, n)
    for i := range users {
        users[i] = NewUser(t)
        users[i].ID = int64(1000 + i)
    }
    return users
}
```

## Mocks

Function-field pattern for flexible mocking:

```go
//go:build testing

package testing

import (
    "context"

    "github.com/zoobz-io/sumatra/models"
)

type MockUsers struct {
    OnGet        func(ctx context.Context, key string) (*models.User, error)
    OnSet        func(ctx context.Context, key string, user *models.User) error
    OnGetByLogin func(ctx context.Context, login string) (*models.User, error)
}

func (m *MockUsers) Get(ctx context.Context, key string) (*models.User, error) {
    if m.OnGet != nil {
        return m.OnGet(ctx, key)
    }
    return &models.User{}, nil  // sensible default
}

func (m *MockUsers) Set(ctx context.Context, key string, user *models.User) error {
    if m.OnSet != nil {
        return m.OnSet(ctx, key, user)
    }
    return nil
}

func (m *MockUsers) GetByLogin(ctx context.Context, login string) (*models.User, error) {
    if m.OnGetByLogin != nil {
        return m.OnGetByLogin(ctx, login)
    }
    return &models.User{}, nil
}
```

## Registry Helpers

```go
//go:build testing

package testing

import (
    "context"
    "testing"

    "github.com/zoobz-io/sum"
    sumtest "github.com/zoobz-io/sum/testing"
    "github.com/zoobz-io/sumatra/contracts"
)

type RegistryOption func(k sum.Key)

func WithUsers(u contracts.Users) RegistryOption {
    return func(k sum.Key) {
        sum.Register[contracts.Users](k, u)
    }
}

func SetupRegistry(t *testing.T, opts ...RegistryOption) context.Context {
    t.Helper()
    sum.Reset()
    k := sum.Start()
    for _, opt := range opts {
        opt(k)
    }
    sum.Freeze(k)
    t.Cleanup(sum.Reset)
    return sumtest.TestContext(t)
}
```

## Usage in Tests

```go
//go:build testing

package handlers

import (
    "testing"

    rtesting "github.com/zoobz-io/rocco/testing"
    vickytest "github.com/zoobz-io/sumatra/testing"
)

func TestGetMe(t *testing.T) {
    user := vickytest.NewUser(t)
    mu := &vickytest.MockUsers{
        OnGet: func(ctx context.Context, key string) (*models.User, error) {
            return user, nil
        },
    }

    _ = vickytest.SetupRegistry(t, vickytest.WithUsers(mu))

    // Test handler...
}
```

## Directory Structure

```
testing/
├── helpers.go      # Fixtures and registry setup
├── mocks.go        # Mock implementations
├── integration/    # Integration tests
│   └── README.md
└── benchmarks/     # Performance benchmarks
    └── README.md
```

## Guidelines

- Always use `//go:build testing` build tag
- Always call `t.Helper()` in helper functions
- Provide sensible defaults in mocks (return empty structs, not errors)
- Use `With*` pattern for registry options
- Clean up with `t.Cleanup(sum.Reset)`
