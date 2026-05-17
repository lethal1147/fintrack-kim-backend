# Auth Feature — Implementation Plan

## Dependency Graph

```
pkg/jwtutil      ─┐
pkg/hashutil     ─┤
pkg/apperror     ─┤
pkg/response     ─┤→ AuthService → AuthHandler → router.go + main.go
domain interfaces─┘
middleware.Auth  ──────────────────────────────→ router.go (guard routes)
```

All dependencies exist. Nothing needs to be created before Task 1.

---

## Phases

### Phase 1 — AuthService (business logic)

**Task A1 — AuthService implementation + unit tests**

Files to create:
- `internal/service/auth_service.go`
- `internal/service/auth_service_test.go`

Acceptance criteria:
- `AuthService` struct implements `AuthServiceInterface` (defined in same file)
- Constructor: `NewAuthService(userRepo, sessionRepo, cfg AuthServiceConfig) *AuthService`
- `AuthServiceConfig` holds `AccessSecret`, `RefreshSecret`, `AccessExpiryMinutes`, `RefreshExpiryDays` (avoids coupling service to `config.Config`)
- **Register:** hashes password (bcrypt), checks duplicate email (→ Conflict), creates User row, creates Session row, returns token pair + user info
- **Login:** finds user by email (unknown email → Unauthorized, same message), verifies bcrypt hash (wrong password → Unauthorized), creates Session row, returns token pair + user info
- **Refresh:** looks up refresh token in DB (not found → Unauthorized), parses JWT (expired/tampered → Unauthorized), returns new access token
- **Logout:** calls `SessionRepository.DeleteByID` on the session found by refresh token; idempotent (not-found is not an error)
- **LogoutAll:** calls `SessionRepository.DeleteAllByUserID`
- **GetProfile:** calls `UserRepository.FindByID`, returns `*UserInfo` (404 on not found)

Unit tests (mock repos — no DB):

| Test | What it verifies |
|---|---|
| `TestRegister_Success` | User created, session created, tokens returned |
| `TestRegister_DuplicateEmail` | `apperror.Conflict` returned when email exists |
| `TestLogin_Success` | Correct credentials → token pair |
| `TestLogin_WrongEmail` | `apperror.Unauthorized` |
| `TestLogin_WrongPassword` | `apperror.Unauthorized` (same error as wrong email) |
| `TestRefresh_Success` | Valid token in DB → new access token |
| `TestRefresh_NotInDB` | `apperror.Unauthorized` |
| `TestRefresh_ExpiredToken` | `apperror.Unauthorized` |
| `TestLogout_Success` | `DeleteByID` called |
| `TestLogout_NotFound` | No error returned (idempotent) |
| `TestLogoutAll_Success` | `DeleteAllByUserID` called with correct userID |
| `TestGetProfile_Success` | Returns `UserInfo` with correct fields |
| `TestGetProfile_NotFound` | `apperror.NotFound` returned |

Verification: `go test ./internal/service/... -v` — all 13 tests green.

---

### ✅ Checkpoint 1 — `go test ./internal/service/...` all green

---

### Phase 2 — AuthHandler + handler tests

**Task A2 — AuthHandler implementation + unit tests**

Files to create:
- `internal/handler/auth_handler.go`
- `internal/handler/auth_handler_test.go`

Acceptance criteria:
- `AuthHandler` struct with `AuthServiceInterface` injected
- Constructor: `NewAuthHandler(svc AuthServiceInterface) *AuthHandler`
- **Register handler:** `ShouldBindJSON` → call service → `response.Created` or `response.Error`
- **Login handler:** bind, extract `User-Agent` + `c.ClientIP()`, call service, respond
- **Refresh handler:** bind, call service, respond
- **Logout handler:** bind, call service (`middleware.ContextUserID` not needed for logout — token identifies session), respond
- **LogoutAll handler:** extract `userID` from context (`middleware.ContextUserID`), call service, respond
- **Me handler:** extract `userID` from context, call `GetProfile`, respond

All 6 handler functions must have complete Swagger annotations.

Binding validation on handlers:
- `Register`: `binding:"required,email"` on email; `binding:"required,min=8"` on password; `binding:"required"` on name → returns 400 on failure
- `Login`: `binding:"required,email"` + `binding:"required"` on password
- `Refresh`: `binding:"required"` on refresh_token
- `Logout`: `binding:"required"` on refresh_token

Unit tests (mock service — no real tokens):

| Test | What it verifies |
|---|---|
| `TestRegisterHandler_Success` | 201, `data.access_token` present |
| `TestRegisterHandler_MissingFields` | 400 on missing body |
| `TestRegisterHandler_ShortPassword` | 400 when password < 8 chars |
| `TestRegisterHandler_Conflict` | 409 when service returns Conflict |
| `TestLoginHandler_Success` | 200, token pair present |
| `TestLoginHandler_InvalidCredentials` | 401 when service returns Unauthorized |
| `TestRefreshHandler_Success` | 200, `data.access_token` present |
| `TestRefreshHandler_InvalidToken` | 401 |
| `TestLogoutHandler_Success` | 200, requires Auth middleware |
| `TestLogoutHandler_Unauthenticated` | 401 (no Bearer token) |
| `TestLogoutAllHandler_Success` | 200, requires Auth middleware |
| `TestMeHandler_Success` | 200, profile fields present |
| `TestMeHandler_Unauthenticated` | 401 (no Bearer token) |

Verification: `go test ./internal/handler/... -v` — all 13 tests green.

---

### ✅ Checkpoint 2 — `go test ./internal/handler/...` all green

---

### Phase 3 — Router wiring + main.go + Swagger

**Task A3 — Wire everything end-to-end**

Files to modify:
- `internal/router/router.go` — add `AuthHandler` to `Handlers` struct; add `/auth` route group
- `cmd/api/main.go` — instantiate `UserRepo`, `SessionRepo`, `AuthService`, `AuthHandler`

Route registration:
```go
auth := r.Group("/auth")
{
    auth.POST("/register", h.Auth.Register)
    auth.POST("/login",    h.Auth.Login)
    auth.POST("/refresh",  h.Auth.Refresh)

    protected := auth.Group("")
    protected.Use(middleware.Auth(cfg.JWTAccessSecret))
    {
        protected.POST("/logout",     h.Auth.Logout)
        protected.POST("/logout-all", h.Auth.LogoutAll)
        protected.GET("/me",          h.Auth.Me)
    }
}
```

main.go additions:
```go
userRepo    := postgres.NewUserRepo(db)
sessionRepo := postgres.NewSessionRepo(db)
authSvc     := service.NewAuthService(userRepo, sessionRepo, service.AuthServiceConfig{...})
authHandler := handler.NewAuthHandler(authSvc)
```

`router.Handlers` updated to include `Auth *handler.AuthHandler`.

After wiring, run: `make swagger` to regenerate `docs/`.

Update router test helper (`router_test_helpers_test.go`) to pass a stub `AuthHandler`.

Verification:
- `go build ./...` compiles with no errors
- `go test ./...` all tests green
- `make swagger` succeeds
- `make dev` starts without panic
- Manual smoke tests:
  - `POST /auth/register` → 201
  - `POST /auth/login` → 200 with tokens
  - `GET /auth/me` (with access token) → 200 with profile
  - `GET /swagger/index.html` → shows 6 auth endpoints

---

### ✅ Final Checkpoint — All DoD items in auth-spec.md §9 checked

---

## File Summary

| File | Action |
|---|---|
| `internal/service/auth_service.go` | Create |
| `internal/service/auth_service_test.go` | Create |
| `internal/handler/auth_handler.go` | Create |
| `internal/handler/auth_handler_test.go` | Create |
| `internal/router/router.go` | Modify — add Auth routes + handler |
| `internal/router/router_test_helpers_test.go` | Modify — stub AuthHandler |
| `cmd/api/main.go` | Modify — wire auth repos + service + handler |
| `docs/` | Regenerate via `make swagger` |

---

## Risk Notes

- **bcrypt in service tests:** `hashutil.Hash` uses cost 12 — tests that exercise Register + Login will be slow (~100ms each). This is expected and acceptable at test-suite scale.
- **Session token storage:** Refresh token is stored as the raw JWT string in `sessions.refresh_token`. This is the existing design from the scaffold.
- **Router test stub:** `router_test_helpers_test.go` creates a `stubHealthSvc`. It will need a corresponding stub for `AuthServiceInterface` or we wire a nil handler for auth (since router tests only hit `/health`).
