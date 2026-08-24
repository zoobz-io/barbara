# handlers

HTTP handlers for the public API surface.

## Purpose

Define HTTP endpoints using rocco's handler pattern. Handlers are the entry point for HTTP requests and orchestrate calls to contracts.

## Pattern

```go
// handlers/users.go
package handlers

import (
    "github.com/zoobz-io/rocco"
    "github.com/zoobz-io/sum"
    "github.com/zoobz-io/sumatra/api/contracts"
    "github.com/zoobz-io/sumatra/api/transformers"
    "github.com/zoobz-io/sumatra/wire"
)

var GetMe = rocco.GET("/me", func(req *rocco.Request[rocco.NoBody]) (wire.UserResponse, error) {
    users := sum.MustUse[contracts.Users](req.Context)

    user, err := users.Get(req.Context, req.Identity.ID())
    if err != nil {
        return wire.UserResponse{}, err
    }

    return transformers.UserToResponse(user), nil
}).WithSummary("Get current user").
   WithDescription("Returns the authenticated user's profile.").
   WithTags("Users").
   WithAuthentication()

var UpdateMe = rocco.PATCH("/me", func(req *rocco.Request[wire.UserUpdateRequest]) (wire.UserResponse, error) {
    users := sum.MustUse[contracts.Users](req.Context)

    user, err := users.Get(req.Context, req.Identity.ID())
    if err != nil {
        return wire.UserResponse{}, err
    }

    transformers.ApplyUserUpdate(req.Body, user)

    if err := users.Set(req.Context, req.Identity.ID(), user); err != nil {
        return wire.UserResponse{}, err
    }

    return transformers.UserToResponse(user), nil
}).WithSummary("Update current user").
   WithTags("Users").
   WithAuthentication()
```

## Handler Registration

```go
// handlers/handlers.go
package handlers

import "github.com/zoobz-io/rocco"

func All() []rocco.Endpoint {
    return []rocco.Endpoint{
        GetMe,
        UpdateMe,
        // ... all handlers
    }
}
```

## Error Definitions

```go
// handlers/errors.go
package handlers

import "github.com/zoobz-io/rocco"

var (
    ErrUserNotFound = rocco.ErrNotFound.WithMessage("user not found")
    ErrMissingQuery = rocco.ErrBadRequest.WithMessage("query parameter required")
)
```

## Chainable Methods

| Method | Purpose |
|--------|---------|
| `.WithSummary()` | OpenAPI summary |
| `.WithDescription()` | OpenAPI description |
| `.WithTags()` | OpenAPI tag grouping |
| `.WithAuthentication()` | Require auth |
| `.WithPathParams()` | Define path variables |
| `.WithQueryParams()` | Document query parameters |
| `.WithErrors()` | Document expected errors |
| `.WithSuccessStatus()` | Override default 200 |

## Streaming (SSE)

For real-time updates, use `rocco.NewStreamHandler`:

```go
var StreamProgress = rocco.NewStreamHandler[rocco.NoBody, wire.ProgressUpdate](
    "progress-stream",
    http.MethodGet,
    "/jobs/{id}/progress",
    func(req *rocco.Request[rocco.NoBody], stream rocco.Stream[wire.ProgressUpdate]) error {
        for {
            select {
            case <-stream.Done():
                return nil // Client disconnected
            case update := <-progress:
                if err := stream.Send(update); err != nil {
                    return err
                }
            }
        }
    },
).WithPathParams("id").
    WithSummary("Stream job progress").
    WithAuthentication()
```

## Guidelines

- Handlers are module-level variables, not methods
- Use `rocco.NoBody` for requests without a body
- Retrieve contracts via `sum.MustUse[contracts.T](req.Context)`
- Use transformers for model ↔ wire conversion
- Keep handler logic minimal - orchestration only
- Define domain-specific errors in `errors.go`
- Register all handlers in the `All()` function
- For streams: always check `stream.Done()` for client disconnect
