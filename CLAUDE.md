# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make dev          # go run ./cmd/api
make build        # go build -o bin/api ./cmd/api
make test         # go test ./...
make test-cover   # generate coverage.out and open HTML report
make lint         # golangci-lint run
make migrate-up   # apply all pending migrations
make migrate-down # roll back one migration
make docker-up    # start Postgres on port 5433
make docker-down  # stop containers
make swagger      # regenerate docs/ from annotations (requires swag in PATH)
```

## Architecture — Layer Rules

```
domain → zero external imports (no GORM, no Gin, no pkg/)
service → depends on domain interfaces only; no HTTP, no DB driver
handler → depends on service interfaces only; no repos, no GORM
pkg/ → no domain knowledge; reusable across any project
```

Never skip a layer. Handlers do not call repos directly. Services do not import Gin.

## File Conventions

- All files: **kebab-case** filenames
- Package per layer: `config`, `domain`, `service`, `handler`, `middleware`, `router`, `database`, `repository/postgres`
- Tests alongside code: `foo.go` → `foo_test.go`
- Integration tests: `foo_integration_test.go` with `//go:build integration` tag
- Hard limit: **250 lines per file** — split into sub-files if exceeded

## Config (Viper)

- Env var `DB_HOST` maps to struct field `DBHost` (no dots in keys)
- All struct fields must have a `SetDefault` entry so `AutomaticEnv` can override them
- Secrets (`JWT_*`) have empty-string defaults; `validate()` rejects empty/placeholder values
- `.env` is gitignored; `.env.example` is committed

## Swagger

**Every handler function must have a swaggo annotation block:**

```go
// FunctionName godoc
// @Summary      One line summary
// @Description  Longer description
// @Tags         group-name
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Router       /path [method]
func (h *Handler) FunctionName(c *gin.Context) {
```

Run `make swagger` after every annotation change. Never hand-edit files in `docs/`.
`SWAGGER_ENABLED=false` disables the `/swagger/*` route in production.

## Error Handling

Use `pkg/apperror` constructors (`NotFound`, `Unauthorized`, `BadRequest`, etc.). Never return raw Go errors to HTTP handlers — wrap with an `AppError` so `response.Error` maps the correct HTTP status.

## State and Response Shape

All success responses:  `{ "success": true, "data": {...} }`
All error responses:    `{ "success": false, "error": { "code": "...", "message": "..." } }`

Use `response.Success(c, data)`, `response.Created(c, data)`, `response.Error(c, err)` — never `c.JSON` directly in handlers.

## Auth Design

- Access token: short-lived JWT (15 min), stateless, validated by `Auth` middleware
- Refresh token: stored in `sessions` table; one row per device/login
- Logout = delete session row; Logout-all = delete all rows for `user_id`
- `middleware.Auth(secret)` injects `userID` into Gin context via `middleware.ContextUserID` key

## pkg/ Packages

| Package | Purpose |
|---|---|
| `apperror` | Typed errors with HTTP status codes |
| `response` | JSON envelope helpers (`Success`, `Created`, `Error`) |
| `jwtutil` | Sign and parse access/refresh JWTs |
| `hashutil` | bcrypt hash/verify (cost ≥ 12) |

## Testing

- Services: test with mock implementations of domain interfaces (no real DB)
- Handlers: test with `httptest.NewRecorder` and mock service structs
- Minimum 80% coverage on `internal/service/` packages
- Run `go test ./...` before every commit
