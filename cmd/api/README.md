# cmd/api

Public API binary entrypoint.

## Purpose

Contains `main.go` with the bootstrap for the public HTTP surface. Uses the `run() error` pattern for clean error handling. One binary per surface: setup shared by every binary lives in `internal/boot`; this file only does what is specific to the public API.

## Bootstrap Sequence

```go
func run() error {
    ctx := context.Background()

    // 1. Shared setup — sum init, common config, infra, stores, model boundaries.
    //    Registry comes back unfrozen.
    rt, err := boot.Init(ctx)
    defer func() { _ = rt.DB.Close() }()

    // 2. Per-surface config
    if err := sum.Config[config.App](ctx, rt.K, nil); err != nil {
        return fmt.Errorf("failed to load app config: %w", err)
    }

    // 3. Register this surface's contracts over the shared stores
    sum.Register[contracts.Users](rt.K, rt.Stores.Users)

    // 4. Register this surface's wire boundaries
    wire.RegisterBoundaries(rt.K)

    // 5. Freeze registry (no more registrations after this)
    sum.Freeze(rt.K)

    // 6. Observability
    otelProviders, err := boot.OTEL(ctx, "barbara-api")
    ap, err := boot.Aperture(ctx, otelProviders)

    // 7. Register handlers and run
    rt.Svc.Handle(handlers.All()...)
    return rt.Svc.Run("", appCfg.Port)
}
```

## Guidelines

- Keep `main()` minimal - just call `run()` and handle the error
- Shared setup belongs in `internal/boot`, not here — new binaries call `boot.Init` and register their own surface
- Use `defer` for cleanup (database connections, etc.) — the caller owns clients returned by boot
- Emit events at startup milestones for observability
- Register all services before calling `sum.Freeze(k)`
