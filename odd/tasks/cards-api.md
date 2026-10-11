# Tasks: cards-api

Plan: [`odd/plans/cards-api.md`](../plans/cards-api.md)

Rules: one task = one reviewable work unit = at least one Conventional Commit on a feature branch.
Check a box only after the outcome and its checks were observed. Record the commit hash as evidence.

## Handoff (updated 2026-10-11)

Status: all 15 tasks done (T01 to T12, including T06b, T06c and T11b). Branch `feat/cards-api`, nothing pushed yet. Remaining work is in the deferred phases and the open review advisories.

How to resume:

1. `git switch feat/cards-api`, then run `gofmt -l .`, `go build ./...`, `go vet ./...`, `go test -count=1 ./...` (all green at this point). Integration tests need Docker: `go test -tags integration ./internal/adapters/repository/`.
2. Read plan decisions D1 to D13 in [`odd/plans/cards-api.md`](../plans/cards-api.md).
3. Local run: `APP_ENV=local DB_HOST=... DB_NAME=... DB_USER=... DB_PASSWORD=... DB_SSLMODE=disable PAN_SALT=... go run ./cmd/api`.
4. Next: pick from the TODO section at the end of this file. Suggested start: Luhn, expiry and brand detection (pure domain, easy to test).

Open points to carry into the next tasks:

- `/health` takes the instance from `POD_NAME`, falling back to the hostname; on ECS Fargate the hostname is the normal source.
- Open review advisories and deferred phases are listed as a checklist in the TODO section at the end of this file.

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
  Decisions: `limit` 1..100 (default 20) and `last4` format enforced here; `400` for invalid json/limit/last4/cursor, `422` for invalid PAN/expiry/expired card, `404` unknown card, unmapped errors become `500` with a fixed message and are logged. `ErrCardAlreadyExists` was left unmapped under the assumption that ids are server-generated; it now answers `409` with slug `card_already_exists` (the unique `pan_hash` can reject a repeated PAN, see T07 and T09). Added `domain.ErrInvalidCursor`. Routing for `/cards:search` vs `/cards/{id}` is now a permanent test.
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

- [x] **T08 Postgres repositories**
  `CardRepository` implementation. Integration tests (build tag `integration`):
  round-trip, duplicate, not-found, rollback.
  Commit: `5e6fe48`. Check: `go test -tags integration ./internal/adapters/repository/` passed against a postgres:16 container (observed 2026-10-10).
  Note: the connection must use `gorm.Config{TranslateError: true}` (wired in T09). Added `gorm.io/driver/postgres` and `testcontainers-go`.

- [x] **T09 Composition root and entry point**
  `service/application.go`, `cmd/api/main.go`, `internal/config` (plan D12), HMAC `PANHasher` adapter, Secrets Manager client, graceful shutdown.
  Check: service starts against a local Postgres and answers `/health`.
  Commits: `a52b17f` (config, hasher, secrets), `1cf833f` (service, main), `315da92` (APP_ENV).
  Observed: `gofmt`, `go build`, `go vet`, `go vet -tags integration`, `go test -count=1 ./...` green; writer saw RED then GREEN for config, hasher, secrets fallback and the JSON error handler.
  Observed (2026-10-10, APP_ENV=local, postgres:16): the service started, `/health` answered 200, create, list, search and an invalid cursor behaved as expected. Not exercised: the `cloud` path against real AWS Secrets Manager.
  Fixed: the duplicate-PAN 500 now answers `409` `card_already_exists` (`feat(ports): answer 409 for a duplicate card`).
  Findings to fix: GORM logs failed INSERTs, including `pan_hash`, outside slog.
  Fixed: every timestamp is UTC end to end; the DSN sets `TimeZone=UTC`, GORM `NowFunc` returns UTC and rows are mapped with `.UTC()` (`fix(repository): return every timestamp in UTC`).
  Refactor (plan D12): `APP_ENV` (`local`|`cloud`, required, no default) replaces the implicit secret-source detection; `secrets.Resolve` removed. `cloud` is fail-closed: an unreadable or blank secret stops startup before the port opens. Env vars: `APP_ENV` (required), `PORT`, `POD_NAME`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_SSLMODE` (default `require`); `local`: `DB_PASSWORD`, `PAN_SALT`; `cloud`: `AWS_REGION`, `DB_PASSWORD_SECRET_ID`, `PAN_SALT_SECRET_ID`.

## Phase 4 — Delivery of this stage

- [x] **T10 Developer docs**
  `README.md`: how to run, architecture summary, endpoints, deferred items, how AI was used (link to `odd/`).
  Commit: 4edf5dd.

- [x] **T11 Observability for CloudWatch**
  Structured JSON logs to stdout (`slog`), a request-logging middleware (method, route, status, latency, request id, never the PAN or `panHash`), and basic request metrics (count, latency, status) for CloudWatch.
  Decide before starting: metrics through CloudWatch Embedded Metric Format in the logs (no extra SDK) or `PutMetricData`. Requirement: "basic logs and metrics in CloudWatch" (requirements file, section A; idea C7).
  Step 1 done (human-approved): `log/slog` logger built once in `main` and injected (`internal/logging`), `TextHandler` or `JSONHandler` by `LOG_FORMAT`, `LOG_LEVEL`, fixed attributes `service`/`env`/`instance`, UTC RFC3339 time, masking of sensitive keys (also in groups), source only at debug. No new dependency.
  Step 2 done: request id (chi `RequestID`) and request log with latency in `internal/ports/middleware`, registered on the router in `ports.NewHandler` (not in `ChiServerOptions`) so unmatched routes (404) are logged too. Route pattern only, never raw path, query or body; `/health` logs at debug, 5xx at error. RED metrics (requests, latency_ms; dimensions method, route, status_class) go through CloudWatch Embedded Metric Format in the same log line (`_aws`), no SDK; they are only meaningful with `LOG_FORMAT=json` (default in cloud).
  Check: logs and metrics emitted by a handler test; no card data in any log line. Commit: 5afb510 (step 2).

- [x] **T11b Request traceability**
  `internal/ports/middleware.RequestID` replaces chi's `RequestID`: it accepts the incoming `X-Request-Id` only when it is 1 to 64 characters of letters, digits, dot, underscore or hyphen, and generates a UUID otherwise. The id is returned in the `X-Request-Id` response header, in the `requestId` field of every error body (contract updated in `api/openapi/cards.yml`) and in the request log, which now also carries the raw `path` (a plain field, not an EMF dimension; the query string is never logged).
  Why: support can find a request from the id a client reports, or from the card id in the path. The client value is validated because it ends up in logs and responses.
  Front-end guidance: send its own `X-Request-Id` per user action, show `requestId` in error messages, and expose the header through `Access-Control-Expose-Headers` if the API is served cross-origin (CORS is not configured in this API yet).
  Check: middleware tests for valid, missing and invalid ids; request log tests for `path` and no query string; `NewHandler` tests for header, echo, `requestId` in the error body and the same id in the access log. Commit: 8112566.
  Follow-up (review R3): the `method` log field and EMF dimension is bounded to the 7 standard methods plus `OTHER`, because the client controls the method and each new value would create a CloudWatch metric series; the real method of an unusual request is not logged. Commit: cc4647e.

- [x] **T12 Route GORM logs through the configured slog logger**
  Pass the logger built in `main` (`internal/logging`) to `service.NewApplication` and set it on the GORM connection through a small adapter that implements `gorm.io/gorm/logger.Interface` over `*slog.Logger`, so every database log line shares format, level, fixed attributes and masking.
  Why: the default GORM logger writes to stderr outside `slog` and prints the failed `INSERT` with its values, including `pan_hash` (finding of the T09 run, 2026-10-10).
  Scope: map GORM levels to slog levels; never print bound parameters (`ParameterizedQueries`); do not log `ErrRecordNotFound` or `ErrDuplicatedKey` as errors (the repository already maps them to domain errors); report slow queries above a threshold as `warn`.
  Check: a unit test with a buffer logger proves no value of a failed insert reaches the output and that not-found is not logged as an error; the T09 manual run shows only `slog` lines. Commit: cdfa39b.
  Note: the `error` text of non-translated Postgres errors is not controlled by us and is covered only by key-based masking.

## TODO (not part of the 15 tasks above)

Optional improvements to review at the end of the project live in [`odd/requirements-and-improvements.md`](../requirements-and-improvements.md), section C.

Deferred phases:

- [ ] **Luhn, expiry and brand detection**: implement `validatePAN`, `validateExpiry` and `detectBrand` in `internal/domain/card.go` with domain tests. Today every PAN and expiry is accepted and `brand` is always `unknown`.
- [ ] **Secure PAN storage**: only the one-way `panHash` exists; reversible storage (KMS encryption or tokenization) is under research by the human.
- [ ] **Idempotency** by `Idempotency-Key` (same Postgres DB; draft in plan section 5).
- [ ] **Dockerfile, ECS Fargate task definition and service, load balancer, RDS, AWS deployment** (target design is in the README diagram).

Open review advisories:

- [ ] `POST /cards` accepts trailing data after the JSON body.
- [ ] `last4` is not checked as digits in `NewCard` and `UnmarshalCardFromDatabase`.
- [ ] `UnmarshalCardFromDatabase` does not validate `createdAt` or `brand`.
- [ ] Domain tests have weak assertions; `app.Application` can be built with zero-value handlers.
- [ ] GORM logger: treat `context.Canceled` as a non-error (`gorm_logger.go`).
- [ ] `request_log_test.go:146`: the assertion that the real method is not logged is weak.
- [ ] `path` field of the access log: revisit if a route with a sensitive path parameter is ever added.

Manual checks pending:

- [ ] Run against a real Postgres and confirm a failed insert logs only `$1, $2...` (nothing proves end to end that GORM applies `ParamsFilter`; integration tests were never run in this phase because Docker was off).
- [ ] Run the `cloud` secrets path against real AWS Secrets Manager.
- [ ] Run the commands in the README (`docker run` and `go run`), which were written but never executed.

Housekeeping:

- [ ] `odd/code-explanations.md` (Spanish study notes) is intentionally untracked; decide whether to commit it.
