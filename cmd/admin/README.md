# admin

Admin API binary entrypoint.

## Purpose

Standalone binary for internal team operations. Runs separately from the public API with different authentication, authorization, and potentially different network exposure.

## Structure

```go
// cmd/admin/main.go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/zoobz-io/rocco"
    "github.com/zoobz-io/sum"
    "github.com/zoobz-io/sumatra/admin/contracts"
    "github.com/zoobz-io/sumatra/admin/handlers"
    "github.com/zoobz-io/sumatra/admin/stores"
    "github.com/zoobz-io/sumatra/config"
)

func main() {
    ctx := context.Background()

    // Initialize registry
    k := sum.Start(ctx)

    // Load config
    cfg, err := sum.Config[config.App](ctx, k, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Create stores
    allStores, err := stores.New(db, renderer)
    if err != nil {
        log.Fatal(err)
    }

    // Register contracts
    sum.Register[contracts.Users](k, allStores.Users)

    // Freeze registry
    sum.Freeze(k)

    // Create router
    router := rocco.New(handlers.All()...)

    // Start server
    log.Printf("Admin API listening on %s", cfg.AdminPort)
    log.Fatal(http.ListenAndServe(cfg.AdminPort, router))
}
```

## Deployment Considerations

- Run on internal network only
- Use different authentication (internal SSO, admin tokens)
- May have stricter rate limiting
- Consider separate monitoring/alerting
- Audit all operations
