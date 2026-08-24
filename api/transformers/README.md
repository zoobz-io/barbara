# transformers

Pure functions for mapping between models and wire types for the public API surface.

## Purpose

Provide a clean separation between internal models and API types. Transformers handle all conversions, keeping handlers focused on orchestration.

## Pattern

```go
// transformers/users.go
package transformers

import (
    "github.com/zoobz-io/sumatra/models"
    "github.com/zoobz-io/sumatra/wire"
)

// UserToResponse transforms a User model to an API response.
func UserToResponse(u *models.User) wire.UserResponse {
    return wire.UserResponse{
        ID:        u.ID,
        Login:     u.Login,
        Email:     u.Email,
        Name:      u.Name,
        AvatarURL: u.AvatarURL,
    }
}

// UsersToResponse transforms a slice of User models to responses.
func UsersToResponse(users []*models.User) []wire.UserResponse {
    result := make([]wire.UserResponse, len(users))
    for i, u := range users {
        result[i] = UserToResponse(u)
    }
    return result
}

// ApplyUserUpdate applies a UserUpdateRequest to a User model.
func ApplyUserUpdate(req wire.UserUpdateRequest, u *models.User) {
    if req.Name != nil {
        u.Name = req.Name
    }
}
```

## Guidelines

- Pure functions only - no side effects, no database calls
- One file per domain entity
- Name pattern: `ModelToResponse`, `ModelsToResponse`, `ApplyModelUpdate`
- Handle nil checks gracefully
- Use `Apply*` prefix for mutation functions that modify models in place
- Keep logic simple - complex transformations may indicate a design issue
