# handlers

HTTP handlers for the admin API surface.

## Purpose

Define HTTP endpoints for internal team operations. Admin handlers provide system-wide visibility, bulk operations, and administrative capabilities not exposed to customers.

## Pattern

```go
// admin/handlers/users.go
package handlers

import (
    "github.com/zoobz-io/rocco"
    "github.com/zoobz-io/sum"
    "github.com/zoobz-io/sumatra/admin/contracts"
    "github.com/zoobz-io/sumatra/admin/transformers"
    "github.com/zoobz-io/sumatra/wire"
)

var ListUsers = rocco.GET("/users", func(req *rocco.Request[rocco.NoBody]) (wire.AdminUserListResponse, error) {
    users := sum.MustUse[contracts.Users](req.Context)

    limit := 50
    offset := 0
    // Parse pagination from query params...

    list, err := users.List(req.Context, limit, offset)
    if err != nil {
        return wire.AdminUserListResponse{}, err
    }

    return transformers.UsersToAdminList(list), nil
}).WithSummary("List all users").
   WithDescription("Returns paginated list of all users in the system.").
   WithTags("Users").
   WithQueryParams("limit", "offset").
   WithAuthentication()

var GetUser = rocco.GET("/users/{id}", func(req *rocco.Request[rocco.NoBody]) (wire.AdminUserResponse, error) {
    users := sum.MustUse[contracts.Users](req.Context)

    user, err := users.Get(req.Context, req.Params.Path["id"])
    if err != nil {
        return wire.AdminUserResponse{}, err
    }

    return transformers.UserToAdminResponse(user), nil
}).WithPathParams("id").
   WithSummary("Get user by ID").
   WithTags("Users").
   WithAuthentication()
```

## Handler Registration

```go
// admin/handlers/handlers.go
package handlers

import "github.com/zoobz-io/rocco"

func All() []rocco.Endpoint {
    return []rocco.Endpoint{
        ListUsers,
        GetUser,
        SearchUsers,
        ImpersonateUser,
        // ... all admin handlers
    }
}
```

## Error Definitions

```go
// admin/handlers/errors.go
package handlers

import "github.com/zoobz-io/rocco"

var (
    ErrUserNotFound    = rocco.ErrNotFound.WithMessage("user not found")
    ErrCannotImpersonate = rocco.ErrForbidden.WithMessage("cannot impersonate this user")
)
```

## Guidelines

- Admin handlers expose system-wide operations
- Include list/search endpoints with pagination
- Include administrative operations (impersonate, suspend, audit)
- Expose more data than public handlers (less masking)
- Keep handler logic minimal - orchestration only
- Register all handlers in the `All()` function
