# testing/benchmarks

Performance benchmarks.

## Purpose

Measure and track performance of critical code paths. Benchmarks help identify regressions and optimization opportunities.

## Running

```bash
make test-bench
```

## Pattern

```go
//go:build testing

package benchmarks

import "testing"

func BenchmarkUserLookup(b *testing.B) {
    // Setup
    ctx := context.Background()
    users := setupUsersStore(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = users.Get(ctx, "1000")
    }
}

func BenchmarkUserLookup_Parallel(b *testing.B) {
    ctx := context.Background()
    users := setupUsersStore(b)

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, _ = users.Get(ctx, "1000")
        }
    })
}
```

## Guidelines

- Use `b.ResetTimer()` after setup
- Use `b.RunParallel()` for concurrent benchmarks
- Include `-benchmem` flag to track allocations
- Compare results over time to detect regressions
- Focus on hot paths and frequently called code
