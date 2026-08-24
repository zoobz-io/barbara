# contracts

Interface definitions for the public API surface.

## Purpose

Define the boundaries between layers. Handlers depend on contracts, stores implement contracts. This enables testing with mocks and keeps dependencies flowing in one direction.

## Pattern

```go
// contracts/users.go
package contracts

import (
    "context"

    "github.com/zoobz-io/sumatra/models"
)

type Users interface {
    // Get retrieves a user by primary key.
    Get(ctx context.Context, key string) (*models.User, error)
    // Set creates or updates a user.
    Set(ctx context.Context, key string, user *models.User) error
    // GetByLogin retrieves a user by login name.
    GetByLogin(ctx context.Context, login string) (*models.User, error)
}
```

## Guidelines

- One interface per domain entity
- First parameter is always `context.Context`
- Return pointers for single entities, slices for lists
- Return `error` as the last return value
- Document each method with a brief comment
- Keep interfaces focused - prefer multiple small interfaces over one large one

## Naming

- Interface names are nouns: `Users`, `Repositories`, `Sessions`
- Method names follow Go conventions: `Get`, `Set`, `List`, `Delete`
- Use `ByX` suffix for lookup methods: `GetByLogin`, `ListByUserID`
