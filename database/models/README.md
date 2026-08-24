# models

Domain models representing core business entities.

## Purpose

Define the shape of data as it exists in storage and throughout the application. Models are the source of truth for entity structure.

## Pattern

```go
// models/user.go
package models

import (
    "context"
    "time"

    "github.com/zoobz-io/sum"
)

type User struct {
    ID          int64     `json:"id" db:"id" constraints:"primarykey"`
    Login       string    `json:"login" db:"login" constraints:"notnull,unique"`
    Email       string    `json:"email" db:"email" constraints:"notnull"`
    Name        *string   `json:"name,omitempty" db:"name"`
    AccessToken string    `json:"-" db:"access_token" store.encrypt:"aes" load.decrypt:"aes"`
    CreatedAt   time.Time `json:"created_at" db:"created_at" default:"now()"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at" default:"now()"`
}

// BeforeSave encrypts sensitive fields before persistence.
func (u *User) BeforeSave(ctx context.Context) error {
    b := sum.MustUse[*sum.Boundary[User]](ctx)
    stored, err := b.Store(ctx, *u)
    if err != nil {
        return err
    }
    *u = stored
    return nil
}

// AfterLoad decrypts sensitive fields after loading.
func (u *User) AfterLoad(ctx context.Context) error {
    b := sum.MustUse[*sum.Boundary[User]](ctx)
    loaded, err := b.Load(ctx, *u)
    if err != nil {
        return err
    }
    *u = loaded
    return nil
}

// Clone returns a deep copy.
func (u User) Clone() User {
    c := u
    if u.Name != nil {
        n := *u.Name
        c.Name = &n
    }
    return c
}
```

## Struct Tags

| Tag | Purpose | Example |
|-----|---------|---------|
| `json` | JSON serialization | `json:"id"` |
| `db` | Database column mapping | `db:"user_id"` |
| `constraints` | Schema constraints | `constraints:"primarykey"` |
| `default` | Database default | `default:"now()"` |
| `store.encrypt` | Encrypt on save | `store.encrypt:"aes"` |
| `load.decrypt` | Decrypt on load | `load.decrypt:"aes"` |

## Lifecycle Hooks

- `BeforeSave(ctx context.Context) error` - Called before insert/update
- `AfterLoad(ctx context.Context) error` - Called after select

## Guidelines

- Use `json:"-"` for fields that should never be serialized (tokens, passwords)
- Use pointer types for optional fields (`*string`, `*time.Time`)
- Implement `Clone()` for models that need copying
- Keep models as pure data structures - business logic belongs in services
- Register boundaries in main.go: `sum.NewBoundary[models.User](k)`
