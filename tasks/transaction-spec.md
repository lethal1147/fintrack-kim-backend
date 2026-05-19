# Transaction Module Spec — Full-Stack

## 1. Objective

Build the transaction CRUD module for FinTrack end-to-end: PostgreSQL persistence, a REST
API with server-side filtering/pagination, and wiring the existing frontend UI to real
data. Currency is **THB (฿)**. Categories are split into separate income and expense lists.
Transactions support an optional note field. The module also exposes a summary endpoint
for dashboard KPIs and chart data.

**Delete is soft** — records are never physically removed. `DELETE /transactions/:id` sets
`deleted_at` to the current timestamp. All queries filter `deleted_at IS NULL`. This
preserves history for future analytics and allows accidental-delete recovery.

**Full CRUD UI** — the transactions page exposes add, edit, and delete row actions.

**Out of scope:** budget integration, import from CSV/bank, recurring-to-transaction
auto-creation, multi-currency support, restore/undelete UI.

---

## 2. Category Design

Categories are type-scoped. The frontend shows the income list when `type = "income"` and
the expense list when `type = "expense"`. The backend stores the raw string and validates
it against the combined set.

### Income categories (7)

| Slug | Display |
|---|---|
| `Salary` | Salary |
| `Freelance` | Freelance |
| `Business` | Business Income |
| `Gift Received` | Gift Received |
| `Investment Returns` | Investment Returns |
| `Rental Income` | Rental Income |
| `Other Income` | Other Income |

### Expense categories (13)

| Slug | Display |
|---|---|
| `Housing` | Housing |
| `Food & Dining` | Food & Dining |
| `Transport` | Transport |
| `Entertainment` | Entertainment |
| `Health` | Health |
| `Shopping` | Shopping |
| `Utilities` | Utilities |
| `Education` | Education |
| `Investment` | Investment *(treat investing as "paying yourself")* |
| `Travel` | Travel |
| `Gifts & Donations` | Gifts & Donations |
| `Subscriptions` | Subscriptions |
| `Other` | Other |

---

## 3. Domain Model

### Transaction struct (Go)

```go
// domain/transaction.go
type TransactionType string

const (
    TransactionIncome  TransactionType = "income"
    TransactionExpense TransactionType = "expense"
)

type Transaction struct {
    ID        string
    UserID    string
    Merchant  string          // payee/source name
    Category  string
    Note      string          // optional free-text
    Date      time.Time
    Amount    float64         // always positive; type distinguishes direction
    Type      TransactionType
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt *time.Time      // nil = active; non-nil = soft-deleted
}
```

### TypeScript type (frontend)

```ts
// lib/api-client.ts (extend existing)
export type Transaction = {
    id: string
    merchant: string
    category: string
    note: string        // "" when absent
    date: string        // "YYYY-MM-DD"
    amount: number
    type: "income" | "expense"
    created_at: string
    updated_at: string
    deleted_at: string | null   // null = active
}

export type TransactionListResponse = {
    transactions: Transaction[]
    total: number
    page: number
    limit: number
    pages: number
}

export type TransactionSummary = {
    total_income: number
    total_expense: number
    net: number
    by_category: Array<{
        category: string
        type: "income" | "expense"
        total: number
        count: number
    }>
    by_month: Array<{
        month: string           // "YYYY-MM"
        income: number
        expense: number
        net: number
    }>
}
```

---

## 4. API Endpoints

All routes are under `/transactions` and require the `Authorization: Bearer <token>` header
(protected by `middleware.Auth`).

| Method | Path | Description |
|---|---|---|
| `GET` | `/transactions` | List (paginated, filterable) |
| `POST` | `/transactions` | Create |
| `GET` | `/transactions/summary` | KPI + chart aggregation |
| `GET` | `/transactions/:id` | Get one |
| `PUT` | `/transactions/:id` | Update |
| `DELETE` | `/transactions/:id` | Delete |

**Note:** `/transactions/summary` must be registered **before** `/:id` in the router to
avoid Gin matching `"summary"` as an ID.

### GET /transactions — query params

| Param | Type | Default | Description |
|---|---|---|---|
| `type` | `income \| expense` | all | Filter by type |
| `category` | string | all | Exact match |
| `search` | string | — | Case-insensitive match on merchant or note |
| `page` | int | 1 | Page number (1-based) |
| `limit` | int | 8 | Items per page (max 100) |
| `from` | `YYYY-MM-DD` | — | Inclusive start date |
| `to` | `YYYY-MM-DD` | — | Inclusive end date |

Response:
```json
{
    "success": true,
    "data": {
        "transactions": [...],
        "total": 42,
        "page": 1,
        "limit": 8,
        "pages": 6
    }
}
```

### POST /transactions — request body

```json
{
    "merchant": "SCB Bank",
    "category": "Salary",
    "note": "March salary",
    "date": "2026-03-25",
    "amount": 42800.00,
    "type": "income"
}
```

Validation:
- `merchant`: required, max 255 chars
- `category`: required, must be in the combined category set
- `note`: optional, max 1000 chars
- `date`: required, valid `YYYY-MM-DD`, not in future (warn, not block)
- `amount`: required, `> 0`
- `type`: required, `income | expense`

### GET /transactions/summary — query params

| Param | Default | Description |
|---|---|---|
| `from` | first of current month | Inclusive start |
| `to` | today | Inclusive end |

Response:
```json
{
    "success": true,
    "data": {
        "total_income": 42800.00,
        "total_expense": 21050.00,
        "net": 21750.00,
        "by_category": [
            { "category": "Food & Dining", "type": "expense", "total": 4200.00, "count": 5 }
        ],
        "by_month": [
            { "month": "2026-03", "income": 42800.00, "expense": 21050.00, "net": 21750.00 }
        ]
    }
}
```

### PUT /transactions/:id

Same body as POST. Returns updated transaction. Returns 404 if not found or belongs to
a different user.

### DELETE /transactions/:id — soft delete

Sets `deleted_at = NOW()`. The row is never physically removed.

Returns `{ "message": "deleted" }`. Returns 404 if the record does not exist or is
already soft-deleted (idempotent from the caller's perspective — call it once and it
disappears from all queries).

---

## 5. Database

### Migration 000003 — create transactions table

```sql
-- 000003_create_transactions.up.sql
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
    deleted_at TIMESTAMPTZ   NULL               -- soft delete; NULL = active
);

CREATE INDEX idx_tx_user_date     ON transactions (user_id, date DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_tx_user_type     ON transactions (user_id, type)      WHERE deleted_at IS NULL;
CREATE INDEX idx_tx_user_category ON transactions (user_id, category)  WHERE deleted_at IS NULL;
```

```sql
-- 000003_create_transactions.down.sql
DROP TABLE IF EXISTS transactions;
```

**Partial indexes** (`WHERE deleted_at IS NULL`) keep index sizes small — soft-deleted
rows are excluded and never scanned for active-data queries.

---

## 6. Backend Layer Plan

### 6a. `domain/transaction.go`

- Extend existing struct with `Note`, `CreatedAt`, `UpdatedAt`, `DeletedAt *time.Time`
- Add `TransactionFilter` struct (pagination + filter params)
- Add `TransactionRepository` interface
- Add `TransactionService` interface

### 6b. `repository/postgres/transaction_repo.go`

Implements `TransactionRepository`:

```go
type TransactionRepository interface {
    Create(ctx context.Context, tx *Transaction) error
    GetByID(ctx context.Context, id, userID string) (*Transaction, error)
    Update(ctx context.Context, tx *Transaction) error
    SoftDelete(ctx context.Context, id, userID string) error  // sets deleted_at
    List(ctx context.Context, userID string, f TransactionFilter) ([]*Transaction, int, error)
    Summary(ctx context.Context, userID string, from, to time.Time) (*TransactionSummary, error)
}
```

- All repo methods add `AND deleted_at IS NULL` to every WHERE clause.
- `SoftDelete` runs `UPDATE transactions SET deleted_at = NOW() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`. Returns `apperror.NotFound` if no row was affected.
- `List` builds a dynamic WHERE clause; uses `ILIKE` for search against `merchant` and `note`.
- `Summary` uses two aggregate queries (one GROUP BY category, one GROUP BY month truncation), both filtering `deleted_at IS NULL`.

### 6c. `service/transaction_service.go`

Implements `TransactionService`:

```go
type TransactionService interface {
    Create(ctx context.Context, userID string, req CreateTransactionRequest) (*Transaction, error)
    Get(ctx context.Context, id, userID string) (*Transaction, error)
    Update(ctx context.Context, id, userID string, req UpdateTransactionRequest) (*Transaction, error)
    Delete(ctx context.Context, id, userID string) error  // delegates to SoftDelete
    List(ctx context.Context, userID string, f TransactionFilter) (*TransactionListResult, error)
    Summary(ctx context.Context, userID string, from, to time.Time) (*TransactionSummary, error)
}
```

- Service validates category membership against the combined income + expense set, and `amount > 0`.
- `Delete` is a thin wrapper — calls `repo.SoftDelete`, propagates the error.
- Delegates all persistence to the repository.

### 6d. `handler/transaction_handler.go`

- `NewTransactionHandler(svc TransactionServiceInterface) *TransactionHandler`
- Each method parses query/body → calls service → calls `response.Success/Created/Error`
- Reads `userID` from Gin context via `middleware.ContextUserID`
- `List` parses query params with defaults (page=1, limit=8)

### 6e. `router/router.go`

```go
tx := protected.Group("/transactions")
{
    tx.GET("",         h.Transaction.List)
    tx.POST("",        h.Transaction.Create)
    tx.GET("/summary", h.Transaction.Summary)   // before :id
    tx.GET("/:id",     h.Transaction.Get)
    tx.PUT("/:id",     h.Transaction.Update)
    tx.DELETE("/:id",  h.Transaction.Delete)
}
```

---

## 7. Frontend Changes

### 7a. `lib/categories.ts` — new file

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

export type IncomeCategory  = (typeof INCOME_CATEGORIES)[number]
export type ExpenseCategory = (typeof EXPENSE_CATEGORIES)[number]
export type TransactionCategory = IncomeCategory | ExpenseCategory
```

### 7b. `lib/string-util.ts` — THB symbol

```ts
// Replace $ with ฿ in formatMoney and formatMoneyFull
formatMoney(v)     → numeral(v).format("0,0").replace(/^/, "฿")
formatMoneyFull(v) → numeral(v).format("0,0.00").replace(/^/, "฿")
```

This changes the currency symbol globally — all pages (dashboard, budget, goals,
transactions) update automatically.

### 7c. `lib/api-client.ts` — transaction API helpers

```ts
export const transactionApi = {
    list(params: TransactionListParams, token: string): Promise<TransactionListResponse>
    create(body: CreateTransactionBody, token: string): Promise<Transaction>
    get(id: string, token: string): Promise<Transaction>
    update(id: string, body: UpdateTransactionBody, token: string): Promise<Transaction>
    delete(id: string, token: string): Promise<void>
    summary(params: SummaryParams, token: string): Promise<TransactionSummary>
}
```

All calls use `credentials: 'include'` and `Authorization: Bearer <token>` from the
auth store.

### 7d. `store/transactions-store.ts` — server-side state

Replace in-memory mock state with API-backed state:

```ts
type TransactionsState = {
    transactions: Transaction[]
    total: number
    page: number
    pages: number
    isLoading: boolean
    error: string | null

    // filter state (drives re-fetch)
    filter: TransactionFilter

    // actions
    fetchTransactions(): Promise<void>
    setFilter(patch: Partial<TransactionFilter>): void
    addTransaction(body: CreateTransactionBody): Promise<void>
    updateTransaction(id: string, body: UpdateTransactionBody): Promise<void>
    deleteTransaction(id: string): Promise<void>
}
```

`setFilter` merges the patch and resets `page` to 1, then calls `fetchTransactions`.

### 7e. `app/(app)/transactions/page.tsx` — wire filters to store

- Remove local `useMemo` filtering (now done server-side)
- Remove local `totalIncome`/`totalExpense` computation — fetch from summary endpoint or
  use the page's own `filtered` totals from the current page's data
- Read `page`, `pages`, `total`, `transactions` from store
- Pass filter changes through `store.setFilter(...)`
- Remove `CATEGORIES` import — use `INCOME_CATEGORIES`/`EXPENSE_CATEGORIES` from
  `lib/categories.ts` for the category filter dropdown
- Hold `editTarget: Transaction | null` in local UI state; pass to `EditTransactionDialog`
- Pass `onEdit={(tx) => setEditTarget(tx)}` and `onDelete={(id) => store.deleteTransaction(id)}`
  to the list/table view components

### 7f. `components/app/transactions/add-transaction-dialog.tsx`

- Replace `CATEGORIES` import with `INCOME_CATEGORIES`/`EXPENSE_CATEGORIES`
- Show the appropriate list based on `type` toggle
- Pass `note` field in the API body
- Change label "Amount ($)" → "Amount (฿)"
- On submit: call `store.addTransaction(body)` (returns promise) — show loading state,
  catch and display errors

### 7g. `components/app/transactions/edit-transaction-dialog.tsx` — new file

Reuses the same form layout as `add-transaction-dialog.tsx` but:
- Accepts `transaction: Transaction | null` prop (null = closed)
- Pre-fills all fields from the existing transaction on open
- On submit: calls `store.updateTransaction(id, body)` — shows loading/error
- Title reads "Edit Transaction"

```ts
type Props = {
    transaction: Transaction | null   // non-null = dialog is open
    onClose: () => void
}
```

### 7h. `components/app/transactions/transaction-list-view.tsx` and `transaction-table-view.tsx`

Both components receive `onEdit` and `onDelete` callbacks from the page, keeping the
components free of store knowledge:

```ts
type Props = {
    paginated: Transaction[]
    onEdit: (tx: Transaction) => void
    onDelete: (id: string) => void
}
```

- Row actions: pencil (edit) and trash (delete) icons, visible on row hover
- Delete triggers a `window.confirm` before calling `onDelete`
- Add category colors for new categories: Investment, Travel, Gifts & Donations,
  Subscriptions, Salary, Freelance, Business, Gift Received, Investment Returns, etc.
- Remove the `Transaction` type import from `lib/mock-data` — import from `lib/api-client`

---

## 8. Currency: THB (฿)

- **Display:** `฿42,800.00` (two decimal places, thousands separator)
- **Storage:** `DECIMAL(12,2)` in PostgreSQL — exact arithmetic, no float drift
- **Go:** `float64` in domain struct (precision sufficient for 12-digit amounts)
- **No change to date logic, time zones, or locale beyond the currency symbol**

The `formatMoney`/`formatMoneyFull` change in `string-util.ts` propagates to all pages
automatically. No page-by-page update needed.

---

## 9. Error Handling

| Scenario | UI | Backend |
|---|---|---|
| Network error | "Something went wrong. Please try again." inline | — |
| Validation failure | Field-level errors from API response | `400 Bad Request` + `code: "validation_error"` |
| Transaction not found | Toast/inline: "Transaction not found" | `404 Not Found` |
| Unauthorized (no token) | Redirect to `/login` via auth store | `401 Unauthorized` |
| Server error | "Something went wrong." | `500` logged, generic message |

Backend: all errors use `pkg/apperror` constructors, surface via `response.Error`.

---

## 10. New Files

| File | Description |
|---|---|
| `finance-tracker-kim-backend/migrations/000003_create_transactions.up.sql` | Create transactions table + indexes |
| `finance-tracker-kim-backend/migrations/000003_create_transactions.down.sql` | Drop transactions table |
| `finance-tracker-kim-backend/internal/repository/postgres/transaction_repo.go` | Postgres CRUD + list + summary |
| `finance-tracker-kim-backend/internal/repository/postgres/transaction_repo_test.go` | Integration tests (build tag) |
| `finance-tracker-kim-backend/internal/service/transaction_service.go` | Business logic |
| `finance-tracker-kim-backend/internal/service/transaction_service_test.go` | Unit tests with mock repo |
| `finance-tracker-kim-backend/internal/handler/transaction_handler.go` | HTTP handlers |
| `finance-tracker-kim-backend/internal/handler/transaction_handler_test.go` | Handler unit tests |
| `finance-tracker-kim/lib/categories.ts` | Typed income/expense category constants |
| `finance-tracker-kim/components/app/transactions/edit-transaction-dialog.tsx` | Edit dialog pre-filled from existing transaction |

## Modified Files

| File | Change |
|---|---|
| `finance-tracker-kim-backend/internal/domain/transaction.go` | Add Note, CreatedAt, UpdatedAt, DeletedAt; add interfaces |
| `finance-tracker-kim-backend/internal/router/router.go` | Register transaction routes |
| `finance-tracker-kim/lib/string-util.ts` | Change $ → ฿ in formatMoney/formatMoneyFull |
| `finance-tracker-kim/lib/api-client.ts` | Add Transaction types + transactionApi |
| `finance-tracker-kim/lib/mock-data.ts` | Remove CATEGORIES (replaced by lib/categories.ts) |
| `finance-tracker-kim/store/transactions-store.ts` | Replace mock state with API state |
| `finance-tracker-kim/app/(app)/transactions/page.tsx` | Wire to store filter/pagination; add editTarget state |
| `finance-tracker-kim/components/app/transactions/add-transaction-dialog.tsx` | Separate categories, note, ฿ label |
| `finance-tracker-kim/components/app/transactions/transaction-list-view.tsx` | Edit + delete row actions, new category colors |
| `finance-tracker-kim/components/app/transactions/transaction-table-view.tsx` | Edit + delete row actions, new category colors |

---

## 11. Definition of Done

- [ ] `POST /transactions` creates a record and returns it with correct JSON shape
- [ ] `GET /transactions` returns paginated results filtered by type/category/search/date range
- [ ] `GET /transactions/summary` returns totals and month breakdown
- [ ] `PUT /transactions/:id` updates only the authenticated user's transaction (no cross-user access)
- [ ] `DELETE /transactions/:id` soft-deletes (sets `deleted_at`); returns 404 if already deleted or not found
- [ ] Soft-deleted transactions are invisible to all list/summary queries
- [ ] Edit dialog pre-fills existing transaction data and saves via PUT
- [ ] Edit row action visible in both list view and table view
- [ ] Currency displays as ฿ throughout the app (dashboard, transactions, budget, goals)
- [ ] Add-transaction dialog shows income categories for income, expense categories for expense
- [ ] Note field saved and displayed in list/table views
- [ ] Delete row action works end-to-end
- [ ] `go test ./...` green (≥ 80% service coverage)
- [ ] `npm run build` clean
- [ ] `make migrate-up` applies migration cleanly; `make migrate-down` rolls it back
