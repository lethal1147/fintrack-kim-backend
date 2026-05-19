# Transaction Module — Todo

## Phase 1: Backend

- [ ] T1 · Domain + Migration — extend `domain/transaction.go`; write SQL migration 000003
- [ ] T2 · Repo + Service — `transaction_repo.go` + `transaction_service.go` + unit tests
- [ ] T3 · Handler + Router — `transaction_handler.go` + router/main.go wiring + handler tests
- [ ] CHECKPOINT A — `go test ./...` green, `make build` clean, manual smoke test

## Phase 2: Frontend

- [ ] T4 · Frontend Foundation — `lib/categories.ts`, ฿ symbol, `transactionApi` in api-client
- [ ] T5 · Store + Page + Add Dialog — API-backed store, wired transactions page, add dialog
- [ ] T6 · Edit Dialog + Row Actions — `edit-transaction-dialog.tsx`, pencil/trash in list+table
- [ ] CHECKPOINT B — full DoD checklist, `npm run build` clean
