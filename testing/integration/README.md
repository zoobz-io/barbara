# testing/integration

Integration tests that test multiple components together.

## Purpose

Test the interaction between real implementations (not mocks). Integration tests typically involve a real database, real storage, etc.

## Running

```bash
make test-integration
```

## Guidelines

- Use real database connections (typically a test database)
- Clean up test data after each test
- Use `t.Parallel()` cautiously - ensure tests don't interfere
- May require Docker or external services running
- Slower than unit tests - run separately in CI
