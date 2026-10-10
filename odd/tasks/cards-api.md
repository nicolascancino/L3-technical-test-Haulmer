# Tasks: cards-api

Plan: [`odd/plans/cards-api.md`](../plans/cards-api.md)

Rules: one task = one reviewable work unit = at least one Conventional Commit on a feature branch.
Check a box only after the outcome and its checks were observed. Record the commit hash as evidence.

## Phase 1 — Skeleton

- [x] **T01 Bootstrap project**
  `go.mod` (module `github.com/nicolascancino/L3-technical-test-Haulmer-`), remove invalid `internal/A.go`,
  create layer folders, `Makefile`, `.gitignore` additions, pin `oapi-codegen` as a Go tool, confirm DB library.
  Check: `go build ./...` and `go vet ./...` pass (observed). Commit: `cc61fa5`.
  Note: DB library = GORM (human decision, see plan D2; dependency added in T07); `make` is not installed on this Windows machine, run the `go` commands directly.

- [x] **T02 Domain model and ports (signatures only)**
  `Card` entity in `card.go` (private fields, `NewCard`, `UnmarshalCardFromDatabase`, getters) with `validatePAN` (Luhn) and
  `validateExpiry` as signatures without logic, called from `NewCard`, domain errors, `CardRepository` interface.
  Check: `go build`, `go vet`, `go test ./internal/domain` pass; `domain` imports only `context`, `errors`, `time` (observed). Commit: `149b901`.
  Decisions: `Card` keeps `last4`, `expMonth`, `expYear` and does not retain the full PAN (storage pending). Listing and `last4` search are read models, defined in `app/query` (T05), not in `CardRepository`. The stubs accept any value until the deferred Luhn/expiry phase.

- [x] **T03 OpenAPI contract**
  `api/openapi/cards.yml` with the 5 operations, `Error` schema, pagination and masked card schemas.
  Check: `go tool oapi-codegen -generate types,chi-server` parses it and yields the 5 `ServerInterface` methods, including `/cards:search` (observed). Commit: `05b9f0e`.
  Decisions: `400` malformed request, `422` business-rule violation (invalid PAN, expired card), `404` unknown card. Responses never include the PAN. `limit` bounds and defaults live in `ports/`, not in the schema.

- [ ] **T04 Code generation**
  `scripts/openapi-http.sh`, `make openapi_http`, commit generated `internal/ports/*.gen.go`.
  Check: generation is reproducible; `/cards:search` and `/cards/{id}` both generate. Commit: _pending_

## Phase 2 — Application and ports

- [ ] **T05 Application layer**
  `app.Application`, `command.CreateCard`, `query.GetCard`, `ListCards`, `SearchCards`.
  Unit tests with in-memory fakes. Commit: _pending_

- [ ] **T06 HTTP port**
  `HttpServer` implementing `ServerInterface`, domain-error -> status mapping, `/health` with instance identity.
  Tests with `httptest` against fake `app` handlers. Commit: _pending_

## Phase 3 — Persistence and wiring

- [ ] **T07 Postgres schema and tables**
  `tables/` row structs with `gorm` tags, schema/migration for `cards`, GORM dependency, `ErrNotFound`/`ErrDuplicate`, cursor encoding.
  Commit: _pending_

- [ ] **T08 Postgres repositories**
  `CardRepository` implementation. Integration tests (build tag `integration`):
  round-trip, duplicate, not-found, rollback.
  Commit: _pending_

- [ ] **T09 Composition root and entry point**
  `service/application.go`, `cmd/api/main.go`, configuration from environment, graceful shutdown.
  Check: service starts against a local Postgres and answers `/health`. Commit: _pending_

## Phase 4 — Delivery of this stage

- [ ] **T10 Developer docs**
  `README.md`: how to run, architecture summary, endpoints, deferred items, how AI was used (link to `odd/`).
  Commit: _pending_

## Deferred (separate phases, not part of this list)

- Idempotency (`Idempotency-Key`, same Postgres DB; draft in plan section 5)
- Luhn and expiry implementation (+ domain tests)
- Secure PAN storage
- AWS Secrets Manager, CloudWatch
- Dockerfile, Kubernetes manifests, AWS deployment
