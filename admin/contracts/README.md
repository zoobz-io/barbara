# contracts

Interface definitions for the admin API surface.

## Purpose

Define the boundaries between layers for admin operations. Admin contracts may expose broader capabilities than public contracts - system-wide access, bulk operations, and internal data.

## Pattern

```go
// admin/contracts/users.go
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
    // List returns all users with pagination.
    List(ctx context.Context, limit, offset int) ([]*models.User, error)
    // Search finds users by criteria.
    Search(ctx context.Context, query string) ([]*models.User, error)
    // Impersonate generates a token for the target user.
    Impersonate(ctx context.Context, targetID string) (string, error)
}
```

## Guidelines

- Admin contracts may expose more methods than public equivalents
- Include bulk operations (List, Search, BatchUpdate)
- Include administrative operations (Impersonate, Audit, Suspend)
- Document each method with a brief comment
- Keep interfaces focused - prefer multiple small interfaces over one large one

## Naming

- Interface names are nouns: `Users`, `Repositories`, `Sessions`
- Method names follow Go conventions: `Get`, `Set`, `List`, `Delete`
- Use `ByX` suffix for lookup methods: `GetByLogin`, `ListByUserID`
