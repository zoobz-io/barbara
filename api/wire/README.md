# wire

Request and response types for the public API surface.

## Purpose

Define the shape of data at the public API boundary. Wire types are what customers send and receive. Public wire types use boundary masking to protect sensitive data (emails, names, etc.).

## Response Pattern

```go
// wire/users.go
package wire

import (
    "context"

    "github.com/zoobz-io/sum"
)

type UserResponse struct {
    ID        int64   `json:"id" description:"User ID" example:"12345"`
    Login     string  `json:"login" description:"Username" example:"octocat"`
    Email     string  `json:"email" description:"Email address" send.mask:"email"`
    Name      *string `json:"name,omitempty" description:"Display name" send.mask:"name"`
    AvatarURL *string `json:"avatar_url,omitempty" description:"Profile image URL"`
}

// OnSend applies boundary masking before the response is marshaled.
func (u *UserResponse) OnSend(ctx context.Context) error {
    b := sum.MustUse[*sum.Boundary[UserResponse]](ctx)
    masked, err := b.Send(ctx, *u)
    if err != nil {
        return err
    }
    *u = masked
    return nil
}

// Clone returns a deep copy.
func (u UserResponse) Clone() UserResponse {
    c := u
    if u.Name != nil {
        n := *u.Name
        c.Name = &n
    }
    return c
}
```

## Request Pattern

```go
import "github.com/zoobz-io/check"

type UserUpdateRequest struct {
    Name *string `json:"name,omitempty" description:"New display name" example:"Jane Doe"`
}

// Validate validates the request using check.
func (r *UserUpdateRequest) Validate() error {
    return check.All(
        check.OptStr(r.Name, "name").MaxLen(255).V(),
    ).Err()
}

// Clone returns a deep copy.
func (r UserUpdateRequest) Clone() UserUpdateRequest {
    c := r
    if r.Name != nil {
        n := *r.Name
        c.Name = &n
    }
    return c
}
```

## Boundary Registration

```go
// wire/boundary.go
package wire

import "github.com/zoobz-io/sum"

func RegisterBoundaries(k sum.Key) error {
    if _, err := sum.NewBoundary[UserResponse](k); err != nil {
        return err
    }
    // Register all wire types with boundaries
    return nil
}
```

## Struct Tags

| Tag | Purpose | Example |
|-----|---------|---------|
| `json` | JSON field name | `json:"user_id"` |
| `description` | OpenAPI description | `description:"User ID"` |
| `example` | OpenAPI example value | `example:"12345"` |
| `send.mask` | Mask before sending | `send.mask:"email"` |
| `send.redact` | Redact before sending | `send.redact:"***"` |
| `receive.hash` | Hash on receive | `receive.hash:"argon2"` |

## Guidelines

- Response types implement `OnSend()` for boundary masking
- Request types implement `Validate()` for validation using check
- Request types implement `OnEntry()` if they need boundary processing (e.g., password hashing)
- Use `description` and `example` tags for OpenAPI docs
- Use pointer types for optional fields
- Implement `Clone()` for all types using boundary processing
- Keep `boundary.go` with `RegisterBoundaries()` function
- Call `wire.RegisterBoundaries(k)` in main.go before `sum.Freeze(k)`
