# Contributing

Thanks for your interest in contributing to the NASA Go SDK.

## Prerequisites

- Go 1.25 or later
- NASA API key (optional; `DEMO_KEY` works for basic testing)

## Development Setup

```bash
git clone https://github.com/peteretelej/nasa.git
cd nasa
go mod download
bash scripts/setup.sh  # installs pre-push hook
```

The pre-push hook depends only on Go. Its first lint run may download `golangci-lint` `v2.11.4` via `go run`.

## Running Tests

Unit tests use fixtures in `testdata/` and do not call live APIs:

```bash
go test ./...
```

Run the same pre-submit checks that CI and the pre-push hook use:

```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run ./...
go test ./...
```

Run integration tests (requires a `NASA_API_KEY` environment variable):

```bash
./scripts/test-integration.sh
```

## Linting

```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run ./...
```

## Pull Request Guidelines

- Describe the change and why it is needed
- Include tests for new functionality
- Run `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run ./...` and `go test ./...` before submitting
- Keep commits focused on a single change

## Code Style

- Follow existing patterns in the codebase
- Pass `context.Context` as the first parameter on all service methods
- Add doc comments on every exported type, function, and method
- Use `httptest` and `testdata/` fixtures for new tests

## Notes

- GitHub Pages must be enabled in repo settings (Settings > Pages > Source > GitHub Actions) for docs deployment.
- After this work merges, `v1.0.0` will be tagged on `main`, making the module indexable on [pkg.go.dev](https://pkg.go.dev/github.com/peteretelej/nasa).
