# FinTrack API — Implementation Plan

## Dependency Graph

```
go.mod / Makefile / docker-compose
        │
        ├── internal/config          (Viper)
        │         │
        │         └── pkg/*          (response, apperror, jwtutil, hashutil)
        │                   │
        │                   └── internal/domain   (pure structs + interfaces)
        │                             │
        │                   ┌─────────┴──────────┐
        │              migrations/           internal/repository/
        │              (SQL files)           (GORM impls of domain interfaces)
        │                                         │
        │                              internal/service/
        │                              (business logic, no HTTP/DB)
        │                                         │
        │                         ┌───────────────┴───────────────┐
        │                    internal/middleware/         internal/handler/
        │                    (cors, logger, auth, rate)   (thin: parse→service→respond)
        │                                         │
        │                              internal/router/
        │                                         │
        │                                   cmd/api/main.go
        │                                         │
        │                                   docs/ (swag init)
        │
        └── CLAUDE.md / README.md
```

**Key constraint:** `internal/domain` must have zero external imports. Everything depends on it; it depends on nothing outside stdlib.

---

## Phase 1 — Project Foundation

**Goal:** Running repo, Docker Postgres, and all tooling in place. No Go code yet.

### Task 1: Repo skeleton + tooling
**Files:** `go.mod`, `.gitignore`, `Makefile`, `.env.example`, `.env`, `docker-compose.yml`

Acceptance criteria:
- `go.mod` declares module `github.com/joakim/fintrack-api`, go 1.22
- `docker-compose.yml` starts Postgres 16 on port 5432 with named volume `fintrack_pgdata`
- `.env.example` lists every variable from the spec (server, db, jwt, oauth, swagger)
- `.env` is gitignored; `.env.example` is not
- Makefile has all targets from spec (dev, build, test, test-cover, lint, migrate-up, migrate-down, docker-up, docker-down, swagger)
- `make docker-up` starts Postgres with no errors

Verification:
```bash
make docker-up
docker-compose ps          # postgres running
docker exec fintrack-db psql -U fintrack -c '\l'  # fintrack db exists
```

---

## Phase 2 — Config + Shared Utilities

**Goal:** Typed config loading and all `pkg/` helpers compile and are tested.

### Task 2: Config (Viper)
**Files:** `internal/config/config.go`

Acceptance criteria:
- `Config` is a flat typed struct (no nested maps)
- Viper reads `.env` file first, then OS env vars override
- Env var `DB_HOST` maps to struct field `DBHost`
- Missing required fields (secrets) cause a startup error with a clear message
- `APP_ENV=production` panics if `DB_SSLMODE=disable`

Verification:
```bash
go build ./internal/config/...
```

### Task 3: `pkg/response` — JSON envelope
**Files:** `pkg/response/response.go`, `pkg/response/response_test.go`

Acceptance criteria:
- `response.Success(c, data)` → `{ "success": true, "data": ... }`
- `response.Error(c, err)` → `{ "success": false, "error": { "code": "...", "message": "..." } }`
- All fields are always present (no `omitempty` on envelope keys)

Verification:
```bash
go test ./pkg/response/...
```

### Task 4: `pkg/apperror` — typed errors
**Files:** `pkg/apperror/apperror.go`, `pkg/apperror/apperror_test.go`

Acceptance criteria:
- `AppError` carries a code string, HTTP status, and human message
- Sentinel constructors: `NotFound`, `Unauthorized`, `Forbidden`, `Conflict`, `Internal`, `BadRequest`
- `apperror.HTTPStatus(err)` returns 500 for unknown errors

Verification:
```bash
go test ./pkg/apperror/...
```

### Task 5: `pkg/jwtutil` — JWT sign/parse
**Files:** `pkg/jwtutil/jwtutil.go`, `pkg/jwtutil/jwtutil_test.go`

Acceptance criteria:
- `SignAccessToken(userID, secret, expiryMinutes)` returns signed JWT string
- `SignRefreshToken(sessionID, secret, expiryDays)` returns signed JWT string
- `ParseToken(token, secret)` returns claims or error
- Expired token returns `apperror.Unauthorized`
- Tampered signature returns `apperror.Unauthorized`

Verification:
```bash
go test ./pkg/jwtutil/...
```

### Task 6: `pkg/hashutil` — bcrypt helpers
**Files:** `pkg/hashutil/hashutil.go`, `pkg/hashutil/hashutil_test.go`

Acceptance criteria:
- `Hash(plain string)` returns bcrypt hash, cost ≥ 12
- `Verify(plain, hash string)` returns nil on match, error on mismatch
- `Verify` is constant-time (bcrypt guarantee)

Verification:
```bash
go test ./pkg/hashutil/...
```

---

## ✅ Checkpoint A
```bash
go test ./pkg/...     # all green
go build ./...        # compiles
```

---

## Phase 3 — Domain Layer

**Goal:** All business entities and repository interfaces defined. Zero external deps.

### Task 7: Domain models + interfaces
**Files:** `internal/domain/user.go`, `internal/domain/session.go`, and stubs for transaction, account, budget, goal, recurring

Acceptance criteria:
- `user.go`: `User` struct matching migration schema + `UserRepository` interface (FindByID, FindByEmail, Create, Update)
- `session.go`: `Session` struct + `SessionRepository` interface (Create, FindByRefreshToken, DeleteByID, DeleteAllByUserID)
- Remaining domain files are stubs with the struct type only (no interfaces yet) — they exist so the folder structure is complete
- Zero imports outside stdlib

Verification:
```bash
go vet ./internal/domain/...
grep -r "github.com\|gin\|gorm" internal/domain/   # must be empty
```

---

## Phase 4 — Migrations + Database

**Goal:** `make migrate-up` creates `users` and `sessions` tables in the running Postgres.

### Task 8: SQL migrations
**Files:** `migrations/000001_create_users.up.sql`, `migrations/000001_create_users.down.sql`, `migrations/000002_create_sessions.up.sql`, `migrations/000002_create_sessions.down.sql`

Acceptance criteria:
- Up migrations match the schema in SPEC §6 exactly
- Down migrations cleanly reverse the up migrations
- `000001` up enables `pgcrypto` extension for `gen_random_uuid()`
- `make migrate-up` exits 0
- `make migrate-down` (run twice) exits 0 and leaves a clean schema

Verification:
```bash
make docker-up
make migrate-up
docker exec fintrack-db psql -U fintrack -c '\dt'   # shows users, sessions, schema_migrations
make migrate-down   # once
make migrate-down   # twice → empty
make migrate-up     # re-apply, should be idempotent
```

### Task 9: Database connection + repositories
**Files:** `internal/database/database.go` (GORM open + ping), `internal/repository/postgres/user_repo.go`, `internal/repository/postgres/session_repo.go`

Acceptance criteria:
- `database.Connect(cfg)` returns a `*gorm.DB` or error
- `APP_ENV=production` sets `DB_SSLMODE=require` implicitly if not already set
- `UserRepository` and `SessionRepository` implement their domain interfaces fully
- Each repo method has a corresponding unit test using `sqlmock` (or interface-based mock)

Verification:
```bash
go test ./internal/repository/...
go build ./internal/database/...
```

---

## ✅ Checkpoint B
```bash
make docker-up && make migrate-up    # tables created
go test ./...                         # all green
```

---

## Phase 5 — HTTP Stack (Health Endpoint)

**Goal:** Full vertical slice from HTTP request to database — `GET /health` returns 200.

### Task 10: Middleware
**Files:** `internal/middleware/cors.go`, `internal/middleware/logger.go`, `internal/middleware/auth.go`, `internal/middleware/rate_limiter.go`

Acceptance criteria:
- `cors.go`: sets `Access-Control-Allow-Origin` to `APP_FRONTEND_ORIGIN`; rejects other origins in production
- `logger.go`: logs method, path, status, latency per request
- `auth.go`: parses `Authorization: Bearer <token>`, injects `userID` into Gin context; stub that returns 501 if called (implementation in auth task)
- `rate_limiter.go`: stub middleware that passes through (TODO comment for Redis implementation)

Verification:
```bash
go build ./internal/middleware/...
```

### Task 11: Health service + handler + router + main
**Files:** `internal/service/health_service.go`, `internal/service/health_service_test.go`, `internal/handler/health_handler.go`, `internal/handler/health_handler_test.go`, `internal/router/router.go`, `cmd/api/main.go`

Acceptance criteria:
- `HealthService.Check()` pings the DB; returns `{ status, version, db }` struct
- Handler has full swaggo annotation block
- `GET /health` returns `200 { "success": true, "data": { "status": "ok", "version": "0.1.0", "db": "ok" } }`
- If DB is unreachable, `db` field is `"error"` and HTTP status is 503
- `main.go` wires config → DB → repos → services → handlers → router → `r.Run()`
- `make dev` starts without errors
- Handler test uses `httptest.NewRecorder` with a mock service — no real DB

Verification:
```bash
make dev &
curl http://localhost:8080/health
# → {"success":true,"data":{"status":"ok","version":"0.1.0","db":"ok"}}
go test ./internal/service/... ./internal/handler/...
```

---

## ✅ Checkpoint C
```bash
make dev &
curl http://localhost:8080/health   # 200
make test                            # all green
```

---

## Phase 6 — Swagger

**Goal:** `/swagger/index.html` loads and shows the health endpoint.

### Task 12: Swagger setup
**Files:** `docs/` (generated), router update for `/swagger/*any`

Acceptance criteria:
- `go install github.com/swaggo/swag/cmd/swag@latest` documented in README
- `@title`, `@version`, `@host`, `@BasePath` annotations in `main.go`
- Health handler has a complete annotation block (Summary, Description, Tags, Produce, Success, Router)
- `make swagger` exits 0 and regenerates `docs/`
- `SWAGGER_ENABLED=true` → `/swagger/index.html` returns 200
- `SWAGGER_ENABLED=false` → `/swagger/*any` returns 404

Verification:
```bash
make swagger
make dev &
curl http://localhost:8080/swagger/index.html   # 200, HTML
```

---

## Phase 7 — Project Docs

**Goal:** Any new contributor can clone and run in under 5 minutes using just the docs.

### Task 13: CLAUDE.md (backend)
**File:** `CLAUDE.md`

Content must cover:
- All Makefile targets
- Layer dependency rules (domain → no external imports, service → interfaces only, handler → service only)
- Swagger annotation requirement (every handler must have one)
- Viper env var convention (`DB_HOST` → `DBHost` struct field)
- Test file naming (`foo_test.go` alongside `foo.go`)
- `pkg/` has no domain knowledge rule
- File size limit (250 lines, same as frontend)

### Task 14: README.md
**File:** `README.md`

Content must cover:
- Project overview (one paragraph)
- Prerequisites (Go 1.22+, Docker, `swag` CLI)
- Quick start (clone → copy `.env.example` → `make docker-up` → `make migrate-up` → `make dev`)
- All Makefile targets with one-line descriptions
- Swagger URL
- Folder structure summary

---

## ✅ Final Checkpoint — Definition of Done

```bash
make docker-up                                    # ✅ Postgres starts
make migrate-up                                   # ✅ Both migrations clean
make dev &                                         # ✅ Server starts
curl localhost:8080/health                        # ✅ 200 ok
curl localhost:8080/swagger/index.html            # ✅ Swagger UI
make swagger                                      # ✅ docs/ regenerated
make test                                         # ✅ all green, ≥80% service coverage
make lint                                         # ✅ zero errors
```

---

## Task Execution Order

```
Task 1  (foundation)
    ↓
Task 2  (config)
Task 3  (pkg/response)      ← parallel
Task 4  (pkg/apperror)      ← parallel
Task 5  (pkg/jwtutil)       ← parallel
Task 6  (pkg/hashutil)      ← parallel
    ↓
✅ Checkpoint A
    ↓
Task 7  (domain)
    ↓
Task 8  (migrations)
Task 9  (db + repos)        ← parallel after Task 7
    ↓
✅ Checkpoint B
    ↓
Task 10 (middleware)
    ↓
Task 11 (health full-stack) ← needs all of the above
    ↓
✅ Checkpoint C
    ↓
Task 12 (swagger)
    ↓
Task 13 (CLAUDE.md)         ← parallel
Task 14 (README.md)         ← parallel
    ↓
✅ Final Checkpoint
```
