# transformers

Pure functions for mapping between models and wire types for the admin API surface.

## Purpose

Provide transformations specific to admin needs. Admin transformers may expose more data than public equivalents - internal IDs, audit fields, raw values without masking.

## Pattern

```go
// admin/transformers/users.go
package transformers

import (
    "github.com/zoobz-io/sumatra/models"
    "github.com/zoobz-io/sumatra/wire"
)

// UserToAdminResponse transforms a User model to an admin API response.
// Exposes more fields than the public transformer.
func UserToAdminResponse(u *models.User) wire.AdminUserResponse {
    return wire.AdminUserResponse{
        ID:        u.ID,
        Login:     u.Login,
        Email:     u.Email,        // Unmasked
        Name:      u.Name,
        AvatarURL: u.AvatarURL,
        CreatedAt: u.CreatedAt,    // Audit field
        UpdatedAt: u.UpdatedAt,    // Audit field
        LastLogin: u.LastLogin,    // Internal data
        Status:    u.Status,       // Internal status
    }
}

// UsersToAdminList transforms a slice of User models to an admin list response.
func UsersToAdminList(users []*models.User) wire.AdminUserListResponse {
    result := wire.AdminUserListResponse{
        Users: make([]wire.AdminUserResponse, len(users)),
    }
    for i, u := range users {
        result.Users[i] = UserToAdminResponse(u)
    }
    return result
}
```

## Guidelines

- Pure functions only - no side effects, no database calls
- One file per domain entity
- Expose more fields than public transformers where appropriate
- Include audit fields (created_at, updated_at)
- Include internal data (last_login, status, flags)
- Skip masking for internal visibility
- Name pattern: `ModelToAdminResponse`, `ModelsToAdminList`
