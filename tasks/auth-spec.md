# Auth Feature Spec

## 1. Objective

Implement local (email/password) authentication for the FinTrack API.
Covers: register, login, token refresh, logout (single session), and logout-all (all devices).
OAuth is out of scope for this task.

---

## 2. Endpoints

### POST /auth/register

**Request:**
```json
{ "email": "user@example.com", "password": "min8chars", "name": "Kim" }
```

**Validation (400 if fails):**
- `email` — required, valid email format
- `password` — required, min 8 characters
- `name` — required, not empty

**Success — 201:**
```json
{
  "success": true,
  "data": {
    "access_token": "<jwt>",
    "refresh_token": "<jwt>",
    "user": { "id": "uuid", "email": "...", "name": "..." }
  }
}
```

**Errors:**
| Code | Reason |
|---|---|
| 400 | Validation failure (missing / invalid field) |
| 409 | Email already registered |
| 500 | Unexpected error |

---

### POST /auth/login

**Request:**
```json
{ "email": "user@example.com", "password": "mypassword" }
```

**Behavior:**
- Looks up user by email
- Verifies bcrypt password hash
- Creates a new `sessions` row (captures `User-Agent` + `X-Forwarded-For`/`RemoteAddr`)
- Returns token pair + user info

**Success — 200:**
```json
{
  "success": true,
  "data": {
    "access_token": "<jwt>",
    "refresh_token": "<jwt>",
    "user": { "id": "uuid", "email": "...", "name": "..." }
  }
}
```

**Errors:**
| Code | Reason |
|---|---|
| 400 | Missing fields |
| 401 | Invalid credentials (same message regardless of whether email or password is wrong — do not reveal which) |
| 500 | Unexpected error |

---

### POST /auth/refresh

**Request:**
```json
{ "refresh_token": "<jwt>" }
```

**Behavior:**
- Looks up the token in `sessions` by value
- Parses JWT to validate signature and expiry
- Issues a new access token (refresh token is NOT rotated — keep session alive)

**Success — 200:**
```json
{
  "success": true,
  "data": { "access_token": "<jwt>" }
}
```

**Errors:**
| Code | Reason |
|---|---|
| 400 | Missing field |
| 401 | Token not found in DB, expired, or tampered |
| 500 | Unexpected error |

---

### POST /auth/logout

**Auth required:** `Authorization: Bearer <access_token>`

**Request:**
```json
{ "refresh_token": "<jwt>" }
```

**Behavior:** Deletes the matching session row. Idempotent — OK if already deleted.

**Success — 200:**
```json
{ "success": true, "data": { "message": "logged out" } }
```

**Errors:**
| Code | Reason |
|---|---|
| 401 | Missing or invalid access token |
| 500 | Unexpected error |

---

### POST /auth/logout-all

**Auth required:** `Authorization: Bearer <access_token>`

**No request body.**

**Behavior:** Deletes all session rows for the authenticated `user_id`.

**Success — 200:**
```json
{ "success": true, "data": { "message": "logged out from all devices" } }
```

**Errors:**
| Code | Reason |
|---|---|
| 401 | Missing or invalid access token |
| 500 | Unexpected error |

---

### GET /auth/me

**Auth required:** `Authorization: Bearer <access_token>`

**No request body.**

**Behavior:** Returns the full profile of the currently authenticated user. Used by the frontend to rehydrate the user store after page reload.

**Success — 200:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "Kim",
    "avatar_url": "",
    "provider": "local",
    "created_at": "2026-01-01T00:00:00Z"
  }
}
```

**Errors:**
| Code | Reason |
|---|---|
| 401 | Missing or invalid access token |
| 404 | User record not found (edge case — account deleted after token issued) |
| 500 | Unexpected error |

---

## 3. Service Interface

```go
// internal/service/auth_service.go

type AuthInput struct {
    Email    string
    Password string
    Name     string
}

type LoginInput struct {
    Email     string
    Password  string
    UserAgent string
    IPAddress string
}

type AuthResponse struct {
    AccessToken  string
    RefreshToken string
    User         UserInfo
}

type UserInfo struct {
    ID        string
    Email     string
    Name      string
    AvatarURL string
    Provider  string
    CreatedAt time.Time
}

type RefreshResponse struct {
    AccessToken string
}

type AuthServiceInterface interface {
    Register(input AuthInput) (*AuthResponse, error)
    Login(input LoginInput) (*AuthResponse, error)
    Refresh(refreshToken string) (*RefreshResponse, error)
    Logout(refreshToken string) error
    LogoutAll(userID string) error
    GetProfile(userID string) (*UserInfo, error)
}
```

---

## 4. Request / Response DTOs

These live in `internal/handler/auth_handler.go` (unexported, local to handler):

```go
type registerRequest struct {
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
    Name     string `json:"name"     binding:"required"`
}

type loginRequest struct {
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

type logoutRequest struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}
```

---

## 5. Implementation Layers

### Repository (already implemented)
- `UserRepository.FindByEmail` — used in register (conflict check) and login
- `UserRepository.Create` — used in register
- `SessionRepository.Create` — used in register + login
- `SessionRepository.FindByRefreshToken` — used in refresh + logout
- `SessionRepository.DeleteByID` — used in logout
- `SessionRepository.DeleteAllByUserID` — used in logout-all

### Service
- `AuthService` implements `AuthServiceInterface`
- Injected deps: `UserRepository`, `SessionRepository`, JWT config (secrets + expiry values)
- No Gin imports, no HTTP types

### Handler
- `AuthHandler` with methods: `Register`, `Login`, `Refresh`, `Logout`, `LogoutAll`
- Extracts `User-Agent` from `c.GetHeader("User-Agent")`
- Extracts IP from `c.ClientIP()`
- Injects `userID` from context via `middleware.ContextUserID` (for logout endpoints)

### Router
- All auth routes under `/auth` prefix
- `Logout`, `LogoutAll`, and `Me` behind `middleware.Auth(cfg.JWTAccessSecret)`

```
POST /auth/register    → no auth
POST /auth/login       → no auth
POST /auth/refresh     → no auth
POST /auth/logout      → Auth middleware required
POST /auth/logout-all  → Auth middleware required
GET  /auth/me          → Auth middleware required
```

---

## 6. Security Requirements

- **Credentials not leaked:** Login returns identical `401 "invalid credentials"` for wrong email and wrong password. No timing side-channel mitigation required at this stage (bcrypt cost already dominates).
- **bcrypt cost ≥ 12:** `hashutil.Hash` already enforces this.
- **JWT secrets from env:** Already enforced by `config.validate()`.
- **No sensitive data in response:** Never return `password_hash` in any response.
- **Session creation captures device info:** `user_agent` and `ip_address` stored for audit trail.

---

## 7. Swagger Annotations

All five handler functions must have complete swaggo annotation blocks. Example:

```go
// Register godoc
// @Summary      Register a new user
// @Description  Creates a user account and returns an access/refresh token pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      registerRequest  true  "Registration payload"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      409   {object}  map[string]interface{}
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
```

Run `make swagger` after all annotations are in place.

---

## 8. Testing Strategy

### Unit tests — service (mock repos, no DB)

| Test | Description |
|---|---|
| `TestRegister_Success` | Creates user, returns token pair |
| `TestRegister_DuplicateEmail` | Returns 409 `apperror.Conflict` |
| `TestRegister_WeakPassword` | Passwords < 8 chars rejected at handler validation (not service) |
| `TestLogin_Success` | Correct credentials → token pair |
| `TestLogin_WrongEmail` | Unknown email → `apperror.Unauthorized` |
| `TestLogin_WrongPassword` | Known email, wrong password → same `apperror.Unauthorized` |
| `TestRefresh_Success` | Valid token in DB → new access token |
| `TestRefresh_NotInDB` | Token valid JWT but not in DB → `apperror.Unauthorized` |
| `TestRefresh_Expired` | Expired token in DB → `apperror.Unauthorized` |
| `TestLogout_Success` | Deletes session row |
| `TestLogoutAll_Success` | Deletes all session rows for user |

### Unit tests — handler (mock service, httptest)

| Test | Description |
|---|---|
| `TestRegisterHandler_Success` | 201 + correct body shape |
| `TestRegisterHandler_InvalidBody` | 400 on missing fields |
| `TestRegisterHandler_Conflict` | 409 on duplicate email |
| `TestLoginHandler_Success` | 200 + correct body shape |
| `TestLoginHandler_InvalidCredentials` | 401 |
| `TestRefreshHandler_Success` | 200 + access_token present |
| `TestRefreshHandler_InvalidToken` | 401 |
| `TestLogoutHandler_Success` | 200 |
| `TestLogoutHandler_Unauthenticated` | 401 (no Bearer token) |
| `TestLogoutAllHandler_Success` | 200 |
| `TestMeHandler_Success` | 200 + profile fields present |
| `TestMeHandler_Unauthenticated` | 401 (no Bearer token) |

---

## 9. Definition of Done

- [ ] All 6 auth endpoints registered in router
- [ ] `POST /auth/register` creates user + session, returns 201 with token pair
- [ ] `POST /auth/login` verifies credentials, creates session, returns 200 with token pair
- [ ] `POST /auth/refresh` issues new access token without rotating refresh token
- [ ] `POST /auth/logout` deletes session row, requires Auth middleware
- [ ] `POST /auth/logout-all` deletes all sessions for user, requires Auth middleware
- [ ] `GET /auth/me` returns authenticated user's profile, requires Auth middleware
- [ ] All endpoints have swaggo annotations; `make swagger` succeeds
- [ ] `go test ./...` passes — service tests use mock repos (no DB required)
- [ ] 80%+ coverage on `internal/service/`
- [ ] `make lint` passes with zero errors
