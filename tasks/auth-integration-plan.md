# Auth Integration — Implementation Plan

## Dependency Graph

```
I1: Backend cookie support  ──────────────────────────────────────────────────┐
I2: Next.js proxy + api-client  ──→  I3: auth-store  ──→  I4: login + guard  ──→ Final
                                                      ──→  I5: register + sidebar ──→ Final
```

I1 and I2 have no dependencies — can be done in either order.
I3 depends on I2. I4 and I5 both depend on I3 and can be done in either order.

---

## Tasks

### I1 — Backend: cookie support

**Files changed:**
- `internal/config/config.go` — add `AppCookieSecure bool` with `SetDefault("APP_COOKIE_SECURE", false)`
- `internal/handler/auth_handler.go` — cookie set/clear; remove body DTOs for Refresh/Logout
- `internal/router/router.go` — move Logout to public group; pass cookie secure flag
- `cmd/api/main.go` — pass `CookieSecure` to `NewAuthHandler`
- `internal/handler/auth_handler_test.go` — update tests for cookie behaviour

**Behaviour changes:**

| Endpoint | Before | After |
|---|---|---|
| `POST /auth/register` | Returns `{access_token, refresh_token, user}` | Sets httpOnly cookie; returns `{access_token, user}` |
| `POST /auth/login` | Returns `{access_token, refresh_token, user}` | Sets httpOnly cookie; returns `{access_token, user}` |
| `POST /auth/refresh` | Reads `{refresh_token}` from JSON body | Reads cookie; no request body |
| `POST /auth/logout` | Reads `{refresh_token}` from JSON body; requires Bearer auth | Reads cookie; clears cookie; no auth required; moved to public route group |

`LogoutAll` and `Me` are unchanged.

**Cookie settings:**
```
Name:     refresh_token
Path:     /
HttpOnly: true
SameSite: Strict
Secure:   cfg.AppCookieSecure (false in dev, true in prod)
MaxAge:   30 * 24 * 60 * 60  (30 days)
```

**`NewAuthHandler` signature change:**
```go
func NewAuthHandler(svc service.AuthServiceInterface, cookieSecure bool) *AuthHandler
```

**Swagger annotation updates:**
- `Refresh`: remove `@Param body`, update description to "reads refresh_token cookie"
- `Logout`: remove `@Param body`, remove `@Security BearerAuth`, update description
- `Register` + `Login`: update `@Success` to reflect new response shape (no refresh_token)

**Acceptance criteria:**
- `go test ./...` passes with ≥ 89 tests green (update test expectations for cookies)
- `go build ./...` compiles
- `make swagger` succeeds
- Manual: `POST /auth/login` with curl → response has `Set-Cookie: refresh_token=...; HttpOnly`

---

### ✅ Checkpoint I1 — `go test ./...` green, cookie set in login/register response

---

### I2 — Frontend: Next.js proxy + API client

**Files changed:**
- `finance-tracker-kim/next.config.ts` — add rewrite proxy
- `finance-tracker-kim/lib/api-client.ts` — new file

**`next.config.ts`:**
```ts
async rewrites() {
    return [
        {
            source: '/api/:path*',
            destination: `${process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'}/:path*`,
        },
    ]
}
```

**`lib/api-client.ts` exports:**
```ts
// Types
export type UserProfile = { id, email, name, avatar_url, provider, created_at }
export type AuthResponse = { access_token: string; user: UserProfile }

// Auth API helpers (all use /api/auth/* via proxy)
export const authApi = {
    register(name, email, password): Promise<AuthResponse>
    login(email, password): Promise<AuthResponse>
    refresh(): Promise<{ access_token: string }>  // sends cookie automatically
    logout(): Promise<void>                        // sends cookie automatically
    logoutAll(accessToken: string): Promise<void>
    me(accessToken: string): Promise<UserProfile>
}
```

All calls use `credentials: 'include'` (sends cookies).
Protected calls (`logoutAll`, `me`) include `Authorization: Bearer <token>` header.
Non-2xx responses throw `ApiError` with `{ code, message }` extracted from response body.

**Acceptance criteria:**
- `npm run build` succeeds in `finance-tracker-kim/`
- `curl http://localhost:3000/api/health` proxies to backend and returns 200
  (requires both `npm run dev` and `make dev` running)

---

### ✅ Checkpoint I2 — `npm run build` clean, proxy forwards /api/* to backend

---

### I3 — Frontend: auth store

**File created:** `finance-tracker-kim/store/auth-store.ts`

**Store shape:**
```ts
type AuthState = {
    user: UserProfile | null
    accessToken: string | null
    isLoading: boolean       // true while refreshAccessToken() is in-flight
    error: string | null

    // computed
    isAuthenticated: boolean  // derived: accessToken !== null

    // actions
    login(email: string, password: string): Promise<void>
    register(name: string, email: string, password: string): Promise<void>
    logout(): Promise<void>
    refreshAccessToken(): Promise<boolean>  // false = cookie gone/expired
    loadUser(): Promise<void>               // populates user via GET /auth/me
    clearError(): void
}
```

**`refreshAccessToken` logic:**
1. Call `authApi.refresh()`
2. On success: set `accessToken`, return `true`
3. On 401/error: clear state, return `false`

**`login` / `register` logic:**
1. Call service fn → get `{ access_token, user }`
2. Set `accessToken` + `user` in store

**`logout` logic:**
1. Call `authApi.logout()` (best-effort, ignore error)
2. Call `authApi.logoutAll(accessToken)` — or just logout (single session)
3. Clear `accessToken` + `user`

Note: Use single-session logout (POST /auth/logout) — user clicked "Sign out" in sidebar.

**Acceptance criteria:**
- `npm run build` succeeds

---

### I4 — Frontend: login form + route guard (vertical slice #1)

**Files changed:**
- `finance-tracker-kim/app/(app)/layout.tsx` — auth guard
- `finance-tracker-kim/app/(auth)/login/page.tsx` — wire form

**`(app)/layout.tsx` pattern:**
```tsx
"use client"

// On mount: refreshAccessToken() → if false → router.replace('/login')
// While loading: show full-page spinner
// When authenticated: render Sidebar + children
```

Uses `useEffect` + `useAuthStore`. Layout must be a Client Component.
After successful refresh, also calls `store.loadUser()` to populate user profile.

**Login page wiring:**
- Add state: `email`, `password`, `isLoading`, `error`
- `onSubmit`: prevent default, call `store.login(email, password)`
- On success: `router.push('/dashboard')`
- On error: set `error` from caught `ApiError.message`; show below submit button
- Show loading spinner on button while `isLoading`
- Google OAuth button: disable (keep UI, make non-functional with `disabled` attr)

**Acceptance criteria:**
- Visit `/dashboard` without login → redirected to `/login`
- Enter valid credentials → redirected to `/dashboard`
- Page reload on `/dashboard` → session restored, stays on `/dashboard`
- Enter wrong credentials → inline error message visible
- Enter wrong credentials → no redirect

---

### ✅ Checkpoint I4 — Full login → dashboard flow works end-to-end

---

### I5 — Frontend: register form + sidebar (vertical slice #2)

**Files changed:**
- `finance-tracker-kim/app/(auth)/register/page.tsx` — wire form
- `finance-tracker-kim/components/app/sidebar.tsx` — real user + logout

**Register page wiring:**
- Add state: `name`, `email`, `password`, `confirmPassword`, `isLoading`, `error`
- Client-side: validate `password === confirmPassword` before submit; show inline error if mismatch
- `onSubmit`: call `store.register(name, email, password)`
- On success: `router.push('/dashboard')`
- On error: show inline error (duplicate email = "An account with this email already exists")
- Google OAuth button: disable (same as login)

**Sidebar changes:**
- Replace hardcoded `K` / `Kim Joakim` / `kim@example.com` with `useAuthStore().user`
- Avatar initials: `stringUtil.initials(user?.name ?? '')` — fallback to `'?'` while loading
- Add logout button below user row (import `IconLogout` from tabler):
  ```tsx
  <button onClick={handleLogout} ...>
    <IconLogout className="size-4 shrink-0" />
    Sign out
  </button>
  ```
  `handleLogout`: call `store.logout()` → `router.replace('/login')`

**Acceptance criteria:**
- Register with new email → redirected to `/dashboard`, sidebar shows correct name
- Register with existing email → inline error "An account with this email already exists"
- Passwords don't match → inline error before submit
- Sidebar shows logged-in user's name and email (not hardcoded values)
- Click "Sign out" → redirected to `/login`, cookie cleared

---

### ✅ Final Checkpoint — All items in auth-integration-spec.md §8 checked

---

## File Summary

| File | Project | Action |
|---|---|---|
| `internal/config/config.go` | backend | Modify — add AppCookieSecure |
| `internal/handler/auth_handler.go` | backend | Modify — cookie set/clear, remove body DTOs |
| `internal/handler/auth_handler_test.go` | backend | Modify — update for cookie behaviour |
| `internal/router/router.go` | backend | Modify — move Logout to public group |
| `cmd/api/main.go` | backend | Modify — pass cookieSecure to handler |
| `docs/` | backend | Regenerate via make swagger |
| `next.config.ts` | frontend | Modify — add rewrite proxy |
| `lib/api-client.ts` | frontend | Create |
| `store/auth-store.ts` | frontend | Create |
| `app/(app)/layout.tsx` | frontend | Modify — auth guard (convert to client component) |
| `app/(auth)/login/page.tsx` | frontend | Modify — wire form |
| `app/(auth)/register/page.tsx` | frontend | Modify — wire form |
| `components/app/sidebar.tsx` | frontend | Modify — real user + logout |

---

## Risk Notes

- **`(app)/layout.tsx` becomes a Client Component**: Next.js App Router allows this, but it
  means children won't benefit from RSC streaming until the auth check resolves. Acceptable
  for a personal app. The loading state prevents content flash.

- **`stringUtil.initials` fallback**: If `user` is null during hydration, avatar shows `?`.
  Add null-safe guard: `user?.name ? stringUtil.initials(user.name) : '?'`.

- **Cookie `SameSite=Strict` + Next.js proxy**: Strict mode only allows the cookie on
  same-site navigations. Since the browser calls `localhost:3000/api/auth/*` (not a
  cross-site request), Strict works correctly here.

- **Existing handler tests**: Tests that check `Refresh` reads from body and `Logout`
  requires `Authorization` header must be updated. Test count may drop temporarily before
  new cookie-based tests are added.
