# config

Configuration types loaded via fig and registered with sum.

## Purpose

Define typed configuration structs that are loaded from environment variables and optionally from a secret provider (e.g., Vault).

## Pattern

```go
// config/app.go
package config

import "errors"

type App struct {
    Port           int    `env:"APP_PORT" default:"8080"`
    SessionSignKey string `env:"SESSION_SIGN_KEY" secret:"app/session-sign-key"`
}

func (c App) Validate() error {
    if c.Port <= 0 {
        return errors.New("port must be positive")
    }
    return nil
}
```

## Struct Tags

| Tag | Purpose | Example |
|-----|---------|---------|
| `env` | Environment variable name | `env:"APP_PORT"` |
| `default` | Default value if not set | `default:"8080"` |
| `secret` | Secret path for sensitive values | `secret:"app/db-password"` |

## Usage in main.go

```go
// Load config and register with sum
if err := sum.Config[config.App](ctx, k, nil); err != nil {
    return fmt.Errorf("failed to load app config: %w", err)
}

// Retrieve from registry
appCfg := sum.MustUse[config.App](ctx)
```

## Guidelines

- One file per config domain (app.go, database.go, storage.go)
- Always implement `Validate() error`
- Add helper methods like `DSN()` for connection strings
- Use `secret` tag for credentials, keys, and tokens
- Avoid nested structs; prefer flat, single-purpose config types (e.g., separate `GitHub` config file rather than nesting inside `App`)
