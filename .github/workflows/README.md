# .github/workflows

GitHub Actions CI/CD workflows.

## Workflows

### ci.yml

Main continuous integration pipeline. Runs on push to main and pull requests.

**Jobs:**
- **test** - Run tests on Go 1.24 and 1.25
- **lint** - Run golangci-lint
- **security** - Run gosec and upload SARIF results
- **coverage** - Generate and upload coverage to Codecov
- **benchmark** - Run benchmarks and upload results
- **ci-complete** - Gate job requiring all others to pass

### Adding New Workflows

Common patterns:

```yaml
# Release workflow
name: Release
on:
  push:
    tags:
      - 'v*'
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - uses: goreleaser/goreleaser-action@v5
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Guidelines

- Use specific versions for actions (v4, not latest)
- Use matrix strategy for testing multiple Go versions
- Upload artifacts for debugging and tracking
- Use `continue-on-error` for non-critical jobs
- Gate merges on `ci-complete` job
