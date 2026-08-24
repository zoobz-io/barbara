# events

Event definitions for observability and inter-component communication.

## Purpose

Define typed events that can be emitted and listened to throughout the application. Events enable loose coupling between components and provide observability hooks.

## Startup Signals

For operational events (startup milestones, health checks), use direct capitan signals:

```go
// events/startup.go
package events

import "github.com/zoobz-io/capitan"

var (
    StartupDatabaseConnected = capitan.NewSignal("app.startup.database.connected", "Database connected")
    StartupStorageConnected  = capitan.NewSignal("app.startup.storage.connected", "Storage connected")
    StartupServicesReady     = capitan.NewSignal("app.startup.services.ready", "All services registered")
    StartupServerListening   = capitan.NewSignal("app.startup.server.listening", "HTTP server listening")
)

// Field keys for additional data.
var (
    StartupPortKey  = capitan.NewIntKey("port")
    StartupErrorKey = capitan.NewErrorKey("error")
)
```

Usage:
```go
capitan.Emit(ctx, events.StartupDatabaseConnected)
capitan.Emit(ctx, events.StartupServerListening, events.StartupPortKey.Field(8080))
```

## Domain Events

For business events with typed payloads, use `sum.Event[T]`:

```go
// events/user.go
package events

import (
    "github.com/zoobz-io/capitan"
    "github.com/zoobz-io/sum"
)

type UserCreatedEvent struct {
    UserID int64
    Login  string
}

// Signals are always defined as vars.
var UserCreatedSignal = capitan.NewSignal("app.user.created", "User created")

var User = struct {
    Created sum.Event[UserCreatedEvent]
}{
    Created: sum.NewInfoEvent[UserCreatedEvent](UserCreatedSignal),
}
```

Emitting:
```go
events.User.Created.Emit(ctx, events.UserCreatedEvent{
    UserID: user.ID,
    Login:  user.Login,
})
```

Listening:
```go
listener := events.User.Created.Listen(func(ctx context.Context, e events.UserCreatedEvent) {
    log.Printf("User created: %s", e.Login)
})
defer listener.Close()
```

## Guidelines

- Use direct capitan signals for operational/startup events
- Use `sum.Event[T]` for domain events with typed payloads
- Group related events in namespace structs
- Keep event structs simple - just the data needed by listeners
- Close listeners when no longer needed
- Document signal descriptions for observability tools
