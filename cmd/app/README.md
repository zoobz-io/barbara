# cmd/app

Public API binary entrypoint.

## Purpose

Contains `main.go` with the application bootstrap logic. Uses the `run() error` pattern for clean error handling.

## Bootstrap Sequence

```go
func run() error {
    ctx := context.Background()

    // 1. Initialize sum service and registry
    svc := sum.New()
    k := sum.Start()

    // 2. Load all configs
    if err := sum.Config[config.App](ctx, k, nil); err != nil {
        return fmt.Errorf("failed to load app config: %w", err)
    }

    // 3. Connect to infrastructure
    db, err := sqlx.Connect("postgres", dbCfg.DSN())
    defer func() { _ = db.Close() }()

    // 4. Create and register stores
    // Import: "github.com/zoobz-io/sumatra/api/stores"
    // Import: "github.com/zoobz-io/sumatra/api/contracts"
    allStores, err := stores.New(db, renderer, bucketProvider)
    sum.Register[contracts.Users](k, allStores.Users)

    // 5. Register boundaries
    sum.NewBoundary[models.User](k)
    wire.RegisterBoundaries(k)

    // 6. Freeze registry (no more registrations after this)
    sum.Freeze(k)

    // 7. Register handlers and run
    // Import: "github.com/zoobz-io/sumatra/api/handlers"
    svc.Handle(handlers.All()...)
    return svc.Run("", appCfg.Port)
}
```

## Guidelines

- Keep `main()` minimal - just call `run()` and handle the error
- Use `defer` for cleanup (database connections, etc.)
- Emit events at startup milestones for observability
- Load all configs before connecting to infrastructure
- Register all services before calling `sum.Freeze(k)`
