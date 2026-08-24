# migrations

SQL database migrations managed by [goose](https://github.com/pressly/goose).

## Naming Convention

```
NNN_description.sql
```

- Three-digit prefix for ordering
- Descriptive name in snake_case

Examples:
- `001_extensions.sql`
- `002_create_users.sql`
- `003_add_user_avatar.sql`

## Format

Each migration contains both Up and Down sections:

```sql
-- +goose Up
CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_login ON users(login);

-- +goose Down
DROP TABLE users;
```

## Running Migrations

```bash
# Via docker-compose (runs automatically on startup)
docker compose up migrate

# Manually
goose -dir migrations postgres "$DATABASE_URL" up
goose -dir migrations postgres "$DATABASE_URL" down
goose -dir migrations postgres "$DATABASE_URL" status
```

## Guidelines

- Never modify existing migrations - create new ones
- Always provide a Down section
- Keep migrations small and focused
- Include indexes in the same migration as the table
