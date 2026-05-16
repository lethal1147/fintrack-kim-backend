# FinTrack API

Go + Gin REST API for the FinTrack personal finance tracker. Provides authentication,
multi-session management, and data endpoints consumed by the `finance-tracker-kim` frontend.

## Prerequisites

| Tool | Version | Install |
|---|---|---|
| Go | 1.22+ | [go.dev](https://go.dev/dl/) |
| Docker | any | [docker.com](https://www.docker.com/) |
| golang-migrate | latest | `brew install golang-migrate` |
| swag | v1.16+ | `go install github.com/swaggo/swag/cmd/swag@latest` |

## Quick Start

```bash
# 1. Clone and enter
git clone <repo> && cd finance-tracker-kim-backend

# 2. Configure environment
cp .env.example .env
# Edit .env — set JWT_ACCESS_SECRET and JWT_REFRESH_SECRET to secure random values

# 3. Start Postgres (port 5433)
make docker-up

# 4. Run migrations
make migrate-up

# 5. Start the server
make dev
# → http://localhost:8080
```

## Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Server + DB status |
| GET | `/swagger/index.html` | Swagger UI (dev only) |

## Makefile Targets

| Target | Description |
|---|---|
| `make dev` | Run development server with live reload |
| `make build` | Compile binary to `bin/api` |
| `make test` | Run all unit tests |
| `make test-cover` | Run tests with HTML coverage report |
| `make lint` | Run golangci-lint |
| `make migrate-up` | Apply all pending SQL migrations |
| `make migrate-down` | Roll back the latest migration |
| `make docker-up` | Start local Postgres container |
| `make docker-down` | Stop containers |
| `make swagger` | Regenerate `docs/` from handler annotations |

## Swagger

Available at `http://localhost:8080/swagger/index.html` when `SWAGGER_ENABLED=true` (default in development).

After changing handler annotations, run:
```bash
make swagger
```

## Project Structure

```
cmd/api/            # Entrypoint — wires config, DB, services, handlers, router
internal/
  config/           # Viper-based typed configuration
  database/         # GORM connection helper
  domain/           # Pure business entities and repository interfaces
  handler/          # Thin Gin handlers (parse → service → respond)
  middleware/        # CORS, logger, auth, rate-limiter
  repository/       # GORM implementations of domain interfaces
  router/           # Route registration
  service/          # Business logic (no HTTP, no DB driver)
migrations/         # golang-migrate SQL up/down files
pkg/
  apperror/         # Typed errors with HTTP status mapping
  hashutil/         # bcrypt helpers
  jwtutil/          # JWT sign/parse
  response/         # Standard JSON envelope
docs/               # Swagger-generated files (do not edit manually)
```

## Environment Variables

See `.env.example` for all variables. Key ones:

| Variable | Description |
|---|---|
| `APP_ENV` | `development` or `production` |
| `APP_PORT` | HTTP listen port (default `8080`) |
| `DB_*` | Postgres connection settings (port `5433` for local) |
| `JWT_ACCESS_SECRET` | Secret for signing access tokens (**required**) |
| `JWT_REFRESH_SECRET` | Secret for signing refresh tokens (**required**) |
| `SWAGGER_ENABLED` | Set `false` to hide Swagger UI in production |
