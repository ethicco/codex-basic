# Backend Guidelines

## Structure

This is a Go service. The executable entry point is `cmd/server/main.go`; internal packages belong in `internal/`; reusable public packages belong in `pkg/`.

## Commands

Run commands from `backend/`:

```bash
go run ./cmd/server
go build ./...
go test ./...
go vet ./...
```

## Code Style and Tests

Format Go code with `gofmt`. Use idiomatic exported names and lowercase package names. Prefer focused packages.

Place tests next to their package with an `_test.go` suffix. Cover new behavior and error paths, and run `go test ./...` and `go vet ./...` after backend changes.
