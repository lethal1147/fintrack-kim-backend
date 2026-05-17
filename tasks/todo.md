# FinTrack API — Task List

## Phase 1 — Foundation
- [ ] **Task 1** — Repo skeleton: `go.mod`, `.gitignore`, `Makefile`, `.env.example`, `.env`, `docker-compose.yml`

## Phase 2 — Config + Shared Utilities
- [ ] **Task 2** — `internal/config/config.go` (Viper, typed struct)
- [ ] **Task 3** — `pkg/response` (JSON envelope + tests)
- [ ] **Task 4** — `pkg/apperror` (typed errors + HTTP status mapping + tests)
- [ ] **Task 5** — `pkg/jwtutil` (sign/parse access + refresh tokens + tests)
- [ ] **Task 6** — `pkg/hashutil` (bcrypt hash/verify + tests)

## ✅ Checkpoint A — `go test ./pkg/...` green, `go build ./...` compiles

## Phase 3 — Domain Layer
- [ ] **Task 7** — `internal/domain/` (User, Session structs + repo interfaces; stubs for transaction, account, budget, goal, recurring)

## Phase 4 — Migrations + Database
- [ ] **Task 8** — `migrations/` (000001 users, 000002 sessions — up + down SQL)
- [ ] **Task 9** — `internal/database/database.go` + `internal/repository/postgres/` (user_repo, session_repo + tests)

## ✅ Checkpoint B — `make docker-up && make migrate-up` creates tables, `go test ./...` green

## Phase 5 — HTTP Stack
- [ ] **Task 10** — `internal/middleware/` (cors, logger, auth stub, rate_limiter stub)
- [ ] **Task 11** — Health full-stack: `health_service` + `health_handler` (with swaggo annotation) + `router` + `main.go`

## ✅ Checkpoint C — `make dev` starts, `curl localhost:8080/health` returns 200

## Phase 6 — Swagger
- [ ] **Task 12** — Swagger: `main.go` annotations, `/swagger/*any` route, `make swagger`, verify UI loads

## Phase 7 — Project Docs
- [ ] **Task 13** — `CLAUDE.md` (backend coding conventions)
- [ ] **Task 14** — `README.md` (project overview + quick start)

## ✅ Final Checkpoint — All DoD items in SPEC.md §11 checked
