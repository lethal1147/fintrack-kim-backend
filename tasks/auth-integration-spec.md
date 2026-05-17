# Auth Integration Spec — Frontend ↔ Backend

## 1. Objective

Wire the FinTrack frontend's auth pages (login, register) and sidebar to the real backend
auth API. Users can create an account, sign in, and sign out. Authenticated state persists
across page reloads using httpOnly cookies.

**Out of scope:** OAuth, settings page security tab, token rotation, mobile sessions view.

---

## 2. Token Strategy

| Token | Storage | Why |
|---|---|---|
| Access token (JWT, 15 min) | Zustand in-memory | Never in DOM; lost on page reload by design |
| Refresh token (JWT, 30 days) | httpOnly cookie | JS cannot read it; CSRF-safe via SameSite=Strict |

### Cross-Origin Cookie Problem (and fix)

Frontend (localhost:3000) calling backend (localhost:8080) is cross-origin. Browsers block
httpOnly `SameSite=Strict` cookies for cross-origin requests. Solution: **Next.js rewrite
proxy**. Frontend calls `/api/auth/*`, Next.js forwards to `http://localhost:8080/auth/*`.
Cookies are set as same-origin (localhost:3000) — no HTTPS needed in dev.

---

## 3. Backend Changes Required

### 3a. CORS — allow credentials

`internal/middleware/cors.go` must set `Access-Control-Allow-Credentials: true` and match
the `Origin` header exactly (wildcard `*` is invalid with `credentials: true`).
Current CORS middleware already does exact-origin matching — just needs the credentials header added.

### 3b. Cookie config added to RouterConfig

```go
type RouterConfig struct {
    // ...existing fields...
    CookieSecure   bool   // true in production (HTTPS)
    CookieDomain   string // empty = default (host only)
}
```

`CookieSecure` is `false` in dev, `true` in production.
`APP_COOKIE_SECURE=false` (new env var with default false).

### 3c. auth_handler.go changes

**Register + Login:** After `issueTokenPair`, set the httpOnly cookie and strip
`refresh_token` from the JSON response body:

```go
// Set httpOnly cookie
http.SetCookie(c.Writer, &http.Cookie{
    Name:     "refresh_token",
    Value:    resp.RefreshToken,
    Path:     "/api/auth",    // matches the Next.js rewrite prefix
    HttpOnly: true,
    Secure:   h.cookieSecure,
    SameSite: http.SameSiteStrictMode,
    MaxAge:   30 * 24 * 60 * 60, // 30 days in seconds
})
// Return only access_token + user in JSON (no refresh_token)
response.Created(c, gin.H{
    "access_token": resp.AccessToken,
    "user":         resp.User,
})
```

**Refresh:** Read refresh token from cookie, not request body:

```go
// POST /auth/refresh — no body required
func (h *AuthHandler) Refresh(c *gin.Context) {
    cookie, err := c.Cookie("refresh_token")
    if err != nil {
        response.Error(c, apperror.Unauthorized("no refresh token"))
        return
    }
    resp, err := h.svc.Refresh(cookie)
    // ...respond with { access_token }
    // Also rotate cookie (reset MaxAge)
}
```

**Logout:** Read cookie + clear it:

```go
func (h *AuthHandler) Logout(c *gin.Context) {
    cookie, err := c.Cookie("refresh_token")
    if err == nil {
        h.svc.Logout(cookie)  // ignore error — cookie might be expired
    }
    // Clear the cookie
    http.SetCookie(c.Writer, &http.Cookie{
        Name:     "refresh_token",
        Path:     "/api/auth",
        HttpOnly: true,
        MaxAge:   -1,
    })
    response.Success(c, gin.H{"message": "logged out"})
}
```

**Logout-all:** Unchanged — still uses `userID` from access token context.

**Me:** Unchanged.

### 3d. New env var

```env
APP_COOKIE_SECURE=false  # set true in production
```

---

## 4. Frontend Changes Required

### 4a. `next.config.ts` — rewrite proxy

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

All API calls use `/api/auth/...` paths. `NEXT_PUBLIC_API_URL` is unset locally (defaults
to localhost:8080) and set in production.

### 4b. `lib/api-client.ts` — typed fetch wrapper

```ts
// Base fetch with credentials (sends cookies)
async function apiFetch<T>(path: string, init?: RequestInit): Promise<T>

// Auth-specific helpers
export const authApi = {
    register(email: string, password: string, name: string): Promise<AuthResponse>
    login(email: string, password: string): Promise<AuthResponse>
    refresh(): Promise<{ access_token: string }>
    logout(): Promise<void>
    logoutAll(): Promise<void>
    me(): Promise<UserProfile>
}
```

- Always `credentials: 'include'` (sends cookies)
- 401 responses → throw `AuthError` (caught by auth store to trigger redirect)
- Non-2xx → throw `ApiError` with `code` and `message` from response body

**Types:**
```ts
export type AuthResponse = {
    access_token: string
    user: UserProfile
}
export type UserProfile = {
    id: string
    email: string
    name: string
    avatar_url: string
    provider: string
    created_at: string
}
```

### 4c. `store/auth-store.ts` — Zustand auth store

```ts
type AuthState = {
    user: UserProfile | null
    accessToken: string | null
    isLoading: boolean

    // derived
    isAuthenticated: boolean

    // actions
    login(email: string, password: string): Promise<void>
    register(email: string, password: string, name: string): Promise<void>
    logout(): Promise<void>
    refreshAccessToken(): Promise<boolean>  // returns false if cookie expired
    loadUser(): Promise<void>               // calls /auth/me, hydrates user
}
```

On app startup (`(app)/layout.tsx`): call `refreshAccessToken()` → if false, redirect to `/login`.
On success: call `loadUser()` to populate `user`.

The `accessToken` is set in the store after every successful login/register/refresh.
It is set in the `Authorization: Bearer` header on every protected API call.

### 4d. `app/(app)/layout.tsx` — route guard

```tsx
"use client"

export default function AppLayout({ children }) {
    const { isAuthenticated, isLoading, refreshAccessToken } = useAuthStore()
    const router = useRouter()

    useEffect(() => {
        refreshAccessToken().then((ok) => {
            if (!ok) router.replace("/login")
        })
    }, [])

    if (isLoading) return <FullPageSpinner />
    if (!isAuthenticated) return null  // avoid flash before redirect

    return (
        <div className="flex min-h-screen bg-background">
            <Sidebar />
            <main className="flex-1 overflow-y-auto">{children}</main>
        </div>
    )
}
```

### 4e. `app/(auth)/login/page.tsx` — wire form

- Add `useState` for `email`, `password`, `error`, `isLoading`
- `onSubmit`: call `authStore.login(email, password)` → on success `router.push('/dashboard')`
- On error: show error message below the submit button
- Disable submit button while loading, show spinner

### 4f. `app/(auth)/register/page.tsx` — wire form

- Add `useState` for `name`, `email`, `password`, `confirmPassword`, `error`, `isLoading`
- Client-side validation: `password === confirmPassword` before submit
- `onSubmit`: call `authStore.register(name, email, password)` → redirect to `/dashboard`
- On error: show error message below the submit button

### 4g. `components/app/sidebar.tsx` — real user + logout

**User info:** Replace hardcoded `K` / `Kim Joakim` / `kim@example.com` with data from
`useAuthStore().user`. Avatar initials via `stringUtil.initials(user.name)`.

**Logout button:** Add below the user info row:

```tsx
<button
    onClick={() => authStore.logout().then(() => router.push('/login'))}
    className="flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-sm w-full
               text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-foreground
               transition-colors"
>
    <IconLogout className="size-4 shrink-0" />
    Sign out
</button>
```

---

## 5. Auth Flow Diagrams

### Login / Register

```
User submits form
  → authApi.login(email, pw) or authApi.register(...)
  → Backend: verify, issue tokens, set refresh_token cookie, return { access_token, user }
  → Store: set accessToken + user in Zustand
  → router.push('/dashboard')
```

### Page Reload (token restore)

```
(app)/layout mounts
  → store.refreshAccessToken()
  → POST /api/auth/refresh  (browser sends cookie automatically)
  → Backend: read cookie, issue new access token
  → Store: set new accessToken
  → store.loadUser() → GET /api/auth/me → set user
  → render children
```

### Logout

```
User clicks "Sign out"
  → store.logout()
  → POST /api/auth/logout  (browser sends cookie; backend deletes session + clears cookie)
  → Store: clear accessToken + user
  → router.replace('/login')
```

---

## 6. Error Handling

| Scenario | UI behaviour |
|---|---|
| Wrong credentials | Inline error below submit: "Invalid email or password" |
| Email already taken | Inline error: "An account with this email already exists" |
| Password < 8 chars | Client-side inline error before submit |
| Passwords don't match (register) | Client-side inline error before submit |
| Network error | Inline error: "Something went wrong. Please try again." |
| Access token expired mid-session | Auto-refresh; if refresh fails → redirect to /login |
| Refresh cookie expired | `refreshAccessToken()` returns false → redirect to /login |

---

## 7. New Files

| File | Description |
|---|---|
| `lib/api-client.ts` | Typed fetch wrapper + auth API helpers |
| `store/auth-store.ts` | Zustand auth store (user, accessToken, actions) |

## Modified Files

| File | Change |
|---|---|
| `next.config.ts` | Add rewrite proxy |
| `app/(app)/layout.tsx` | Add auth guard + isLoading state |
| `app/(auth)/login/page.tsx` | Wire form to auth store |
| `app/(auth)/register/page.tsx` | Wire form to auth store |
| `components/app/sidebar.tsx` | Real user info + logout button |
| `finance-tracker-kim-backend/internal/middleware/cors.go` | Add credentials header |
| `finance-tracker-kim-backend/internal/handler/auth_handler.go` | Cookie set/clear |
| `finance-tracker-kim-backend/internal/router/router.go` | Pass cookie config |
| `finance-tracker-kim-backend/internal/config/config.go` | Add APP_COOKIE_SECURE |

---

## 8. Definition of Done

- [ ] `POST /api/auth/register` creates account, sets cookie, redirects to `/dashboard`
- [ ] `POST /api/auth/login` signs in, sets cookie, redirects to `/dashboard`
- [ ] Page reload on `/dashboard` restores session via cookie (no flash to login)
- [ ] "Sign out" in sidebar calls logout, clears cookie, redirects to `/login`
- [ ] Sidebar shows real user name + initials from auth store
- [ ] Unauthenticated visit to any `/(app)/` route redirects to `/login`
- [ ] Form validation errors (wrong pw, duplicate email) shown inline
- [ ] `go test ./...` still green after backend changes
- [ ] `npm run build` succeeds after frontend changes
