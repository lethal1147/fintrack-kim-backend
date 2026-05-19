# Transaction Module — Implementation Plan

Spec: `tasks/transaction-spec.md`
Stack: Go (Gin + GORM + Postgres) backend · Next.js 16 (App Router + Zustand) frontend

---

## Dependency Graph

```
T1: Domain + Migration
 ├─→ T2: Repo + Service (all CRUD)
 │       └─→ T3: Handler + Router wiring
 │                   └─ CHECKPOINT A (backend green)
 │
 └─→ T4: Frontend Foundation (categories, ฿, api-client)
             └─→ T5: Store + Page + Add Dialog (list/create flow)
                         └─→ T6: Edit Dialog + Row Actions (full CRUD UI)
                                     └─ CHECKPOINT B (full stack DoD)
```

T4 can begin as soon as T1 is done (no live API needed — types are sufficient to write
the client and build cleanly). T5 and T6 require T3 to be live for manual testing.

---

## Phase 1 — Backend

### T1 · Domain + Migration

**What:** Extend `domain/transaction.go` with the full struct, interfaces, and filter
types. Write the SQL migration files.

**Files touched:**
- `internal/domain/transaction.go` (modify)
- `migrations/000003_create_transactions.up.sql` (new)
- `migrations/000003_create_transactions.down.sql` (new)

**Domain changes:**
```go
type Transaction struct {
    ID        string
    UserID    string
    Merchant  string
    Category  string
    Note      string
    Date      time.Time
    Amount    float64
    Type      TransactionType
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt *time.Time      // nil = active
}

type TransactionFilter struct {
    Type     string  // "" = all
    Category string  // "" = all
    Search   string  // matches merchant or note (ILIKE)
    From     string  // YYYY-MM-DD, "" = unset
    To       string  // YYYY-MM-DD, "" = unset
    Page     int     // 1-based, default 1
    Limit    int     // default 8, max 100
}

type TransactionListResult struct {
    Transactions []*Transaction
    Total        int
    Page         int
    Limit        int
    Pages        int
}

type CategoryStat struct {
    Category string
    Type     TransactionType
    Total    float64
    Count    int
}

type MonthStat struct {
    Month   string  // "YYYY-MM"
    Income  float64
    Expense float64
    Net     float64
}

type TransactionSummaryResult struct {
    TotalIncome  float64
    TotalExpense float64
    Net          float64
    ByCategory   []CategoryStat
    ByMonth      []MonthStat
}

type TransactionRepository interface {
    Create(ctx context.Context, tx *Transaction) error
    GetByID(ctx context.Context, id, userID string) (*Transaction, error)
    Update(ctx context.Context, tx *Transaction) error
    SoftDelete(ctx context.Context, id, userID string) error
    List(ctx context.Context, userID string, f TransactionFilter) ([]*Transaction, int, error)
    Summary(ctx context.Context, userID string, from, to time.Time) (*TransactionSummaryResult, error)
}

type TransactionServiceInterface interface {
    Create(ctx context.Context, userID string, req CreateTransactionRequest) (*Transaction, error)
    Get(ctx context.Context, id, userID string) (*Transaction, error)
    Update(ctx context.Context, id, userID string, req UpdateTransactionRequest) (*Transaction, error)
    Delete(ctx context.Context, id, userID string) error
    List(ctx context.Context, userID string, f TransactionFilter) (*TransactionListResult, error)
    Summary(ctx context.Context, userID string, from, to time.Time) (*TransactionSummaryResult, error)
}

type CreateTransactionRequest struct {
    Merchant string
    Category string
    Note     string
    Date     string  // YYYY-MM-DD
    Amount   float64
    Type     TransactionType
}

type UpdateTransactionRequest = CreateTransactionRequest
```

**Migration (up):**
```sql
CREATE TABLE transactions (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    merchant   VARCHAR(255)  NOT NULL,
    category   VARCHAR(100)  NOT NULL,
    note       TEXT          NOT NULL DEFAULT '',
    date       DATE          NOT NULL,
    amount     DECIMAL(12,2) NOT NULL CHECK (amount > 0),
    type       VARCHAR(10)   NOT NULL CHECK (type IN ('income','expense')),
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ   NULL
);

CREATE INDEX idx_tx_user_date     ON transactions (user_id, date DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_tx_user_type     ON transactions (user_id, type)      WHERE deleted_at IS NULL;
CREATE INDEX idx_tx_user_category ON transactions (user_id, category)  WHERE deleted_at IS NULL;
```

**Acceptance criteria:**
- `go build ./...` passes after domain changes
- `make docker-up && make migrate-up` creates the table with all 11 columns
- `make migrate-down` drops the table cleanly
- `make migrate-up` again succeeds (idempotent via migrate versioning)

**Verification:**
```bash
cd finance-tracker-kim-backend
make docker-up
make migrate-up
# psql: \d transactions  →  confirm schema
make migrate-down
make migrate-up
go build ./...
```

---

### T2 · Repository + Service (all CRUD + Summary)

**What:** Implement the Postgres repo and the service layer. Unit-test the service with
a mock repo; write integration test stubs (build-tagged, not run in CI by default).

**Files touched:**
- `internal/repository/postgres/transaction_repo.go` (new)
- `internal/repository/postgres/transaction_repo_test.go` (new, build tag `integration`)
- `internal/service/transaction_service.go` (new)
- `internal/service/transaction_service_test.go` (new)

**Repo — key implementation notes:**
- GORM model `transactionModel` maps to `transactions` table; `DeletedAt` is `*time.Time`
- All query methods chain `.Where("deleted_at IS NULL")` — **never** use GORM's
  built-in soft-delete (`gorm.DeletedAt`) since we control the column name
- `SoftDelete`: `db.Model(&transactionModel{}).Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).Update("deleted_at", time.Now())` — check `RowsAffected == 0` → `apperror.NotFound`
- `List`: builds WHERE from `TransactionFilter`, uses ILIKE for search, `COUNT(*)` for total, LIMIT/OFFSET for page
- `Summary`: two separate queries — one `GROUP BY category, type`, one `GROUP BY date_trunc('month', date)`; combine results in Go

**Service — key implementation notes:**
- `VALID_CATEGORIES` set = union of income + expense categories from spec (20 total)
- `Create`/`Update`: validate category membership + `amount > 0`; parse `date` string → `time.Time`
- `Delete`: thin wrapper around `repo.SoftDelete`
- `Summary`: default `from` = first of current month, `to` = today when zero values passed

**Service tests (unit, mock repo):**
- `TestCreate_Valid` — happy path, repo.Create called once
- `TestCreate_InvalidCategory` — returns validation error
- `TestCreate_ZeroAmount` — returns validation error
- `TestGet_NotFound` — repo returns NotFound, service propagates
- `TestUpdate_Valid` — repo.Update called with correct fields
- `TestDelete_CallsSoftDelete` — repo.SoftDelete called, error propagated
- `TestList_DefaultPagination` — page=0 normalised to 1, limit=0 to 8
- `TestSummary_DefaultDateRange` — from/to zero values → current month defaults

**Acceptance criteria:**
- `go test ./internal/service/...` passes, ≥ 80% coverage on `transaction_service.go`
- `go test ./...` green (no regressions in auth/health)

**Verification:**
```bash
cd finance-tracker-kim-backend
go test ./internal/service/... -v -cover
go test ./...
```

---

### T3 · Handler + Router (all 6 endpoints)

**What:** HTTP handlers for all endpoints, router wiring, main.go wiring.

**Files touched:**
- `internal/handler/transaction_handler.go` (new)
- `internal/handler/transaction_handler_test.go` (new)
- `internal/router/router.go` (modify — add Transaction to Handlers, register routes)
- `cmd/api/main.go` (modify — instantiate repo, service, handler)

**Handler — key notes:**
- Constructor: `NewTransactionHandler(svc TransactionServiceInterface) *TransactionHandler`
- Every handler reads `userID` via `middleware.ContextUserID`; returns 401 if missing
- `List`: parse query params with defaults (page=1, limit=8, clamp limit ≤ 100)
- `Summary`: parse `from`/`to` as `YYYY-MM-DD`; pass zero `time.Time` if absent (service applies defaults)
- `Delete`: returns `response.Success(c, gin.H{"message": "deleted"})` on success
- All handlers have swaggo annotation blocks

**Router changes:**
```go
// router.go — protected group (after Auth middleware)
tx := protected.Group("/transactions")
{
    tx.GET("",         h.Transaction.List)
    tx.POST("",        h.Transaction.Create)
    tx.GET("/summary", h.Transaction.Summary)   // MUST be before /:id
    tx.GET("/:id",     h.Transaction.Get)
    tx.PUT("/:id",     h.Transaction.Update)
    tx.DELETE("/:id",  h.Transaction.Delete)
}
```

**RouterConfig** — no new fields needed (transactions reuse `JWTAccessSecret` already there).

**main.go additions:**
```go
txRepo    := postgres.NewTransactionRepo(db)
txSvc     := service.NewTransactionService(txRepo)
txHandler := handler.NewTransactionHandler(txSvc)

// pass txHandler to router.Handlers
```

**Handler tests:**
- `TestCreateTransaction_Valid` — POST body → 201, response contains id
- `TestCreateTransaction_InvalidCategory` — 400
- `TestCreateTransaction_MissingField` — 400
- `TestListTransactions_DefaultPage` — GET → 200, pagination fields present
- `TestListTransactions_TypeFilter` — ?type=expense filters correctly
- `TestGetTransaction_NotFound` — 404
- `TestUpdateTransaction_Valid` — 200, updated fields returned
- `TestDeleteTransaction_Valid` — 200, `{"message":"deleted"}`
- `TestDeleteTransaction_NotFound` — 404
- `TestSummary_Valid` — 200, all summary fields present
- `TestTransaction_Unauthenticated` — no token → 401 on any protected route

**Acceptance criteria:**
- `go test ./...` green
- `make build` succeeds
- `make swagger` regenerates docs without errors

**Verification:**
```bash
cd finance-tracker-kim-backend
go test ./...
make build
make swagger
```

---

### ✅ CHECKPOINT A — Backend complete

Before moving to frontend, verify:
- `go test ./...` green, ≥ 80% service coverage
- `make build` clean
- `make migrate-up` clean
- Manual smoke: start server, create a transaction via curl, verify it appears in list,
  soft-delete it, verify it no longer appears in list but row still exists in DB

```bash
# Start server
make docker-up && make migrate-up && make dev

# Smoke test (in another terminal)
TOKEN="<access_token_from_login>"
curl -s -X POST http://localhost:8080/transactions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"merchant":"Test","category":"Salary","date":"2026-05-01","amount":1000,"type":"income"}' | jq .

curl -s http://localhost:8080/transactions \
  -H "Authorization: Bearer $TOKEN" | jq .data.total

# Soft delete check
TX_ID="<id from above>"
curl -s -X DELETE http://localhost:8080/transactions/$TX_ID \
  -H "Authorization: Bearer $TOKEN"
curl -s http://localhost:8080/transactions \
  -H "Authorization: Bearer $TOKEN" | jq .data.total  # should be 0
```

---

## Phase 2 — Frontend

### T4 · Frontend Foundation (categories, ฿, API client)

**What:** The non-network parts of the frontend: category constants, currency symbol
change, typed API client helpers, mock-data cleanup. No store or page changes yet.

**Files touched:**
- `lib/categories.ts` (new)
- `lib/string-util.ts` (modify)
- `lib/api-client.ts` (modify — add Transaction types + transactionApi)
- `lib/mock-data.ts` (modify — remove `CATEGORIES` export, keep everything else)
- `app/(app)/transactions/page.tsx` (modify — update CATEGORIES import to categories.ts)
- `components/app/transactions/add-transaction-dialog.tsx` (modify — update import)

**`lib/categories.ts`:**
```ts
export const INCOME_CATEGORIES = [
    "Salary", "Freelance", "Business", "Gift Received",
    "Investment Returns", "Rental Income", "Other Income",
] as const

export const EXPENSE_CATEGORIES = [
    "Housing", "Food & Dining", "Transport", "Entertainment",
    "Health", "Shopping", "Utilities", "Education",
    "Investment", "Travel", "Gifts & Donations", "Subscriptions", "Other",
] as const

export const ALL_CATEGORIES = [...INCOME_CATEGORIES, ...EXPENSE_CATEGORIES] as const

export type IncomeCategory  = (typeof INCOME_CATEGORIES)[number]
export type ExpenseCategory = (typeof EXPENSE_CATEGORIES)[number]
export type TransactionCategory = IncomeCategory | ExpenseCategory
```

**`lib/string-util.ts` — ฿ symbol:**
```ts
formatMoney(v)     → numeral(v).format("0,0").replace(/^/, "฿")
formatMoneyFull(v) → numeral(v).format("0,0.00").replace(/^/, "฿")
```

**`lib/api-client.ts` additions:**
- Add `Transaction`, `TransactionListResponse`, `TransactionSummary` types
- Add `CreateTransactionBody`, `UpdateTransactionBody`, `TransactionListParams`, `SummaryParams` types
- Add `transactionApi` object with `list`, `create`, `get`, `update`, `delete`, `summary`

**`lib/mock-data.ts`:**
- Remove the `CATEGORIES` and `Category` exports (now in `lib/categories.ts`)
- Leave `Transaction` type for now — it will be replaced in T5 once the store is wired

**Acceptance criteria:**
- `npm run build` clean with zero TypeScript errors
- `฿` symbol appears on dashboard, budget, and goals pages (visual check in browser)
- Category filter dropdown on transactions page shows the updated combined list

**Verification:**
```bash
cd finance-tracker-kim
npm run build
npm run dev  # check ฿ on dashboard
```

---

### T5 · Store + Page + Add Dialog (list/create flow)

**What:** Replace the mock Zustand store with real API calls. Wire the transactions page
to use server-side state. Update the add dialog to call the API.

**Files touched:**
- `store/transactions-store.ts` (rewrite)
- `app/(app)/transactions/page.tsx` (modify — server-side state + filter wiring)
- `components/app/transactions/add-transaction-dialog.tsx` (modify — async, categories, ฿)
- `components/app/transactions/transaction-list-view.tsx` (modify — type import, new category colors)
- `components/app/transactions/transaction-table-view.tsx` (modify — type import, new category colors)

**Store shape:**
```ts
type FilterState = {
    type: "all" | "income" | "expense"
    category: string    // "" = all
    search: string
    from: string        // YYYY-MM-DD or ""
    to: string
    page: number
    limit: number
}

type TransactionsState = {
    transactions: Transaction[]
    total: number
    pages: number
    isLoading: boolean
    error: string | null
    filter: FilterState

    fetchTransactions(): Promise<void>
    setFilter(patch: Partial<FilterState>): void   // merges, resets page to 1
    addTransaction(body: CreateTransactionBody): Promise<void>
    updateTransaction(id: string, body: UpdateTransactionBody): Promise<void>
    deleteTransaction(id: string): Promise<void>
}
```

- `fetchTransactions` reads `accessToken` from auth store; calls `transactionApi.list`
- `setFilter` merges patch, resets `filter.page` to 1, calls `fetchTransactions`
- `addTransaction` calls `transactionApi.create`, then `fetchTransactions` to refresh

**Page changes:**
- Remove `useMemo` filtering and local `sorted`/`filtered` logic
- Remove local `totalPages`, `safePage` — use `store.pages`, `store.filter.page`
- `totalIncome` / `totalExpense` summary strip: compute over current-page `transactions`
  (for the strip above the list) or leave as-is; does not need a full summary call
- Call `store.fetchTransactions()` in a `useEffect([], [])` on mount
- Wire each filter input to `store.setFilter({...})`
- Pass `store.filter.page` to `TransactionPagination`; on page change call `store.setFilter({ page: n })`

**Add dialog changes:**
- Use `INCOME_CATEGORIES` / `EXPENSE_CATEGORIES` instead of `CATEGORIES`
- On type toggle: reset `category` state to `""`
- On submit: call `store.addTransaction(body)` — `isLoading` from store disables button
- Show inline error from `store.error` if the call fails

**List/table view changes:**
- Import `Transaction` from `lib/api-client` instead of `lib/mock-data`
- Expand `CATEGORY_COLORS` with new categories:
  ```ts
  Salary: "bg-emerald-600", Freelance: "bg-emerald-500",
  Business: "bg-teal-600", "Gift Received": "bg-pink-400",
  "Investment Returns": "bg-cyan-500", "Rental Income": "bg-sky-600",
  "Other Income": "bg-emerald-400",
  Investment: "bg-indigo-500", Travel: "bg-amber-600",
  "Gifts & Donations": "bg-rose-400", Subscriptions: "bg-purple-400",
  ```

**Acceptance criteria:**
- `npm run build` clean
- Transactions page loads real data from the API (requires backend running)
- Add a transaction → it appears in the list on the next render
- Filter by type/category/search → list updates via server-side filtering
- Pagination works across server-returned pages

**Verification:**
```bash
cd finance-tracker-kim
npm run build
npm run dev   # with backend running
# manual: open /transactions, add a transaction, filter it
```

---

### T6 · Edit Dialog + Delete Row Actions (full CRUD UI)

**What:** Add the edit dialog (new file) and wire edit + delete row actions to both list
and table view components.

**Files touched:**
- `components/app/transactions/edit-transaction-dialog.tsx` (new)
- `components/app/transactions/transaction-list-view.tsx` (modify — row actions)
- `components/app/transactions/transaction-table-view.tsx` (modify — row actions)
- `app/(app)/transactions/page.tsx` (modify — editTarget state, dialog mount)

**Edit dialog:**
- Props: `transaction: Transaction | null`, `onClose: () => void`
- `open = transaction !== null`
- On open: populate all form fields from `transaction` via `useEffect([transaction], ...)`
- On submit: calls `store.updateTransaction(transaction.id, body)` → `onClose()`
- Shows inline error from `store.error` if the call fails
- Identical form layout to add dialog (type toggle, merchant, amount, date, category, note)

**List view row actions:**
```tsx
// Shown on row hover (group/peer hover pattern)
<div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
    <button onClick={() => onEdit(tx)} title="Edit">
        <IconPencil className="size-3.5" />
    </button>
    <button onClick={() => { if (window.confirm("Delete this transaction?")) onDelete(tx.id) }} title="Delete">
        <IconTrash className="size-3.5 text-destructive" />
    </button>
</div>
```

**Table view row actions:**
- Extra `<td>` column at the far right with the same pencil + trash buttons
- Add `w-16` column header (empty `<th>`) in the header row

**Page changes:**
```tsx
const [editTarget, setEditTarget] = useState<Transaction | null>(null)

// Pass to list/table view:
onEdit={(tx) => setEditTarget(tx)}
onDelete={(id) => store.deleteTransaction(id)}

// Add dialog mount:
<EditTransactionDialog
    transaction={editTarget}
    onClose={() => setEditTarget(null)}
/>
```

**Acceptance criteria:**
- `npm run build` clean
- Clicking pencil opens edit dialog pre-filled with all transaction fields
- Saving the edit updates the row in the list without full page reload
- Clicking trash + confirming removes the row immediately (optimistic or after refetch)
- Soft-deleted records are gone from the list; raw DB row still has `deleted_at` set

**Verification:**
```bash
cd finance-tracker-kim
npm run build
npm run dev   # with backend running
# manual: edit a transaction, verify changes; delete, verify gone from list
# psql: SELECT id, deleted_at FROM transactions ORDER BY created_at DESC LIMIT 5;
#   → deleted row has non-null deleted_at
```

---

### ✅ CHECKPOINT B — Full Stack Definition of Done

Work through the spec's DoD checklist:

- [ ] `POST /transactions` creates a record and returns it with correct JSON shape
- [ ] `GET /transactions` returns paginated results filtered by type/category/search/date range
- [ ] `GET /transactions/summary` returns totals and month breakdown
- [ ] `PUT /transactions/:id` updates only the authenticated user's transaction
- [ ] `DELETE /transactions/:id` soft-deletes; returns 404 if already deleted or not found
- [ ] Soft-deleted transactions invisible to list/summary queries; row preserved in DB
- [ ] Edit dialog pre-fills existing transaction data and saves via PUT
- [ ] Edit row action visible in both list view and table view
- [ ] Currency displays as ฿ throughout the app (dashboard, transactions, budget, goals)
- [ ] Add-transaction dialog shows income categories for income, expense categories for expense
- [ ] Note field saved and displayed in list/table views
- [ ] Delete row action works end-to-end
- [ ] `go test ./...` green (≥ 80% service coverage)
- [ ] `npm run build` clean
- [ ] `make migrate-up` applies cleanly; `make migrate-down` rolls back

---

## Summary

| Task | Layer | Depends on | Key output |
|---|---|---|---|
| T1 | Domain + Migration | — | `domain/transaction.go`, SQL migration |
| T2 | Repo + Service | T1 | All CRUD methods, service unit tests |
| T3 | Handler + Router | T2 | 6 HTTP endpoints, main.go wiring |
| CHECKPOINT A | — | T3 | Backend smoke-tested |
| T4 | FE Foundation | T1 | Categories, ฿, API client types |
| T5 | Store + Page | T3, T4 | API-backed store, wired page + add dialog |
| T6 | Edit + Delete UI | T5 | Edit dialog, row actions |
| CHECKPOINT B | — | T6 | Full DoD checklist |
