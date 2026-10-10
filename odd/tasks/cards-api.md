# Tasks: cards-api

Plan: [`odd/plans/cards-api.md`](../plans/cards-api.md)

Rules: one task = one reviewable work unit = at least one Conventional Commit on a feature branch.
Check a box only after the outcome and its checks were observed. Record the commit hash as evidence.

## Handoff (updated 2026-10-10)

Status: T01 to T06, T06b and T06c done and reviewed (8 of 12). Branch `feat/cards-api`, nothing pushed yet. Next: **T07**.

How to resume:

1. `git switch feat/cards-api`, then run `gofmt -l .`, `go build ./...`, `go vet ./...`, `go test -count=1 ./...` (all green at this point).
2. Read plan decisions D1 to D13 in [`odd/plans/cards-api.md`](../plans/cards-api.md).
3. Start T07 below. It adds the GORM and Postgres driver dependencies.

Open points to carry into the next tasks:

- T07/T08: the reader returns `domain.ErrInvalidCursor` for a bad cursor and `domain.ErrCardNotFound` for an id that is not a valid UUID (plan D8).
- T09: resolve `/health` instance from `POD_NAME`, falling back to the hostname; set `ErrorHandlerFunc` on the generated handler so wrapper errors use the JSON error style.
- Review advisories not yet handled: `POST /cards` accepts trailing data after the JSON body; `last4` is not checked as digits in `NewCard`/`UnmarshalCardFromDatabase`; `UnmarshalCardFromDatabase` does not validate `createdAt` or `brand`; domain tests have weak assertions; `app.Application` can be built with zero-value handlers.
- Deferred phases (not in this list): Luhn and expiry implementation, secure PAN storage (human is researching), idempotency, Secrets Manager, CloudWatch, Dockerfile, ECS Fargate, AWS deployment.

Workflow reminders: generate API code with `./scripts/openapi-http.sh cards internal/ports ports` (no `make`); review one task by starting with `baseRef` set to the full sha of the previous commit; the review lens models live in `~/.pi/gentle-ai/models.json`; a candidate with a shell script is high risk and needs 4 reviewer runs, so ask first.

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

- [x] **T04 Code generation**
  `scripts/openapi-http.sh`, `make openapi_http`, commit generated `internal/ports/*.gen.go`.
  Check: regenerating yields identical checksums; a throwaway router test confirmed `/cards:search?last4=` and `/cards/{id}` dispatch to `SearchCards` and `GetCard` (observed, probe removed). Commit: `17668e5`.
  Note: without `make`, run `./scripts/openapi-http.sh cards internal/ports ports` directly. `go mod tidy` added chi and the oapi-codegen runtime.

## Phase 2 — Application and ports

- [x] **T05 Application layer**
  `app.Application`, `command.CreateCard`, `query.GetCard`, `ListCards`, `SearchCards`.
  Unit tests with in-memory fakes. Commit: `58d39b0`.
  Check: writer observed RED then GREEN; parent re-ran `gofmt`, `go build`, `go vet`, `go test -count=1 ./...` and verified `app` imports only `command`/`query`, which import only `domain` (observed).
  Decisions: `CreateCard` validates through the domain and discards the PAN (secure storage pending). Queries use their own `CardReader` read model in `app/query`. `limit` bounds and `last4` format are enforced in `ports/` (T06), handlers only delegate. Cursor encoding belongs to the reader (T07).

- [x] **T06 HTTP port**
  `HttpServer` implementing `ServerInterface`, domain-error -> status mapping, `/health` with instance identity.
  Tests with `httptest` against fake `app` handlers. Commit: `43194b7`.
  Check: writer observed RED then GREEN; parent re-ran `gofmt`, `go build`, `go vet`, `go test -count=1 ./...` and read `http.go` (observed). `ports` imports only `app`, `command`, `query`, `domain`.
  Decisions: `limit` 1..100 (default 20) and `last4` format enforced here; `400` for invalid json/limit/last4/cursor, `422` for invalid PAN/expiry/expired card, `404` unknown card, unmapped errors become `500` with a fixed message and are logged. `ErrCardAlreadyExists` stays unmapped (ids are server-generated). Added `domain.ErrInvalidCursor`. Routing for `/cards:search` vs `/cards/{id}` is now a permanent test.
  Known limit: `invalid_expiry` and `card_expired` are tested by injecting the errors through the fake repository, because `validateExpiry` is still a stub.
  Follow-up `7dd8d1d`: a body over 1 MiB now answers `413 payload_too_large` (declared in the contract, regenerated, tested RED then GREEN).

## Phase 3 — Persistence and wiring

- [x] **T06b Extend the card model (plan D10)**
  Add `cardHolderName`, `brand`, `panHash` and `updatedAt` to `domain.Card`, `NewCard`, `UnmarshalCardFromDatabase` and getters; `detectBrand` as a signature-only function.
  Update `cards.yml` (request: `cardHolderName`; response: `cardHolderName`, `brand`, `updatedAt`; never `panHash`), regenerate, add the `PANHasher` port (plan D11) with a fake in tests, adapt `command.CreateCard` (computes `panHash`), `query` read model and `ports` mapping, with tests.
  Search (plan D13): add optional `brand`, `cardHolderName`, `expMonth`, `expYear`, `limit` and `cursor` to `cards:search` in the contract, regenerate, extend `query.SearchCards` and the `CardReader` signature, validate in `ports`, with tests.
  Must land before T07 so the table is designed once.
  Part 1 (model fields, `PANHasher`, contract fields): commit `0165f94`; reviewed, approved. Advisory for later: `UnmarshalCardFromDatabase` does not validate `brand`.
  Part 2 (search filters and pagination): commit `e979f65`. Follow-up for T09: when building the generated handler, set `ErrorHandlerFunc` so wrapper errors (missing `last4`, non-integer `limit`) answer the JSON error style instead of `text/plain`.

- [x] **T06c Simplify CreateCard dependencies (plan D6, D8)**
  Remove the injected `newID` and `now` from `NewCreateCardHandler`: generate the id with `uuid.NewV7()` and read `time.Now().UTC()` inside `Handle`; adapt tests and callers; add `google/uuid` to `go.mod`.
  Commit: `895f4ac`.

- [x] **T07 Postgres schema and tables**
  `tables/` row structs with `gorm` tags, schema/migration for `cards` (`id` column of type `uuid`, plan D8), GORM dependency, `ErrNotFound`/`ErrDuplicate`, cursor handling.
  Check: writer observed RED then GREEN; parent re-ran `gofmt`, `go build`, `go vet`, `go test -count=1 ./...` (all green). Commit: `4a61466`.
  Decisions: schema is one embedded `schema.sql` applied with `tables.Migrate` (no migration tool). `pan_hash` is `UNIQUE`, so a repeated PAN is rejected even with another expiry. Indexes: `(last4, id)` and `lower(card_holder_name) text_pattern_ops`. Cursor is the id of the last row of the page; pages are ordered by `id` (UUIDv7, plan D8), human-approved over `(created_at, id)`. Mapping and cursor live in `repository` (not `tables`, which must not import `domain`).
  Gap: `gorm.io/driver/postgres` is not in `go.mod` yet (`go mod tidy` drops it until imported); T08 adds it when it opens the connection.

- [ ] **T08 Postgres repositories**
  `CardRepository` implementation. Integration tests (build tag `integration`):
  round-trip, duplicate, not-found, rollback.
  Commit: `5e6fe48` (code and integration tests written, box stays open).
  Pending: run `go test -tags integration ./internal/adapters/repository/` with Docker up; the SQL was never executed against Postgres. Unit and `go vet -tags integration` are green.
  Note: the connection must use `gorm.Config{TranslateError: true}` (wired in T09). Added `gorm.io/driver/postgres` and `testcontainers-go`.

- [ ] **T09 Composition root and entry point**
  `service/application.go`, `cmd/api/main.go`, `internal/config` (plan D12), HMAC `PANHasher` adapter, Secrets Manager client, graceful shutdown.
  Check: service starts against a local Postgres and answers `/health`.
  Commits: `a52b17f` (config, hasher, secrets), `1cf833f` (service, main). Box stays open.
  Observed: `gofmt`, `go build`, `go vet`, `go vet -tags integration`, `go test -count=1 ./...` green; writer saw RED then GREEN for config, hasher, secrets fallback and the JSON error handler.
  Pending: start the server against Postgres and call `/health`; the AWS Secrets Manager path was never exercised.
  Refactor (plan D12): `APP_ENV` (`local`|`cloud`, required, no default) replaces the implicit secret-source detection; `secrets.Resolve` removed. `cloud` is fail-closed: an unreadable or blank secret stops startup before the port opens. Env vars: `APP_ENV` (required), `PORT`, `POD_NAME`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_SSLMODE` (default `require`); `local`: `DB_PASSWORD`, `PAN_SALT`; `cloud`: `AWS_REGION`, `DB_PASSWORD_SECRET_ID`, `PAN_SALT_SECRET_ID`.

## Phase 4 — Delivery of this stage

- [ ] **T10 Developer docs**
  `README.md`: how to run, architecture summary, endpoints, deferred items, how AI was used (link to `odd/`).
  Commit: _pending_

## Deferred (separate phases, not part of this list)

Optional improvements to review at the end of the project live in [`odd/requirements-and-improvements.md`](../requirements-and-improvements.md), section C.

- Idempotency (`Idempotency-Key`, same Postgres DB; draft in plan section 5)
- Luhn and expiry implementation (+ domain tests)
- Secure PAN storage
- AWS Secrets Manager, CloudWatch
- Dockerfile, ECS Fargate task definition and service, AWS deployment
