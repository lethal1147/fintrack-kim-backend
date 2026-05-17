# Auth Feature — Task List

## Phase 1 — AuthService
- [ ] **A1** — `internal/service/auth_service.go` + `auth_service_test.go` (13 unit tests, mock repos)

## ✅ Checkpoint 1 — `go test ./internal/service/...` all green

## Phase 2 — AuthHandler
- [ ] **A2** — `internal/handler/auth_handler.go` + `auth_handler_test.go` (13 unit tests, mock service)

## ✅ Checkpoint 2 — `go test ./internal/handler/...` all green

## Phase 3 — Wiring + Swagger
- [ ] **A3** — Update `router.go` (auth routes), update `main.go` (wire repos + service + handler), run `make swagger`

## ✅ Final Checkpoint — All items in auth-spec.md §9 checked
