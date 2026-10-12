# Plan: Credit Card API (cards-api)

> Working document produced with an AI coding agent (Pi + el Gentleman harness) following
> Organic Driven Development: the human sets scope and decisions, the agent explores,
> proposes, implements task by task and verifies. Decisions below are the human's unless noted.

## 1. Goal

REST API in Go to create and query credit cards.

| Method | Path                       | Purpose                                              |
| :----- | :------------------------- | :--------------------------------------------------- |
| POST   | `/cards`                   | Create a card (idempotent via `Idempotency-Key`)     |
| GET    | `/cards/{id}`              | Get a card by id                                     |
| GET    | `/cards?limit=&cursor=`    | List cards, cursor pagination                        |
| GET    | `/cards:search?last4=&...` | Search by last 4 digits, optional filters, paginated |
| GET    | `/health`                  | Liveness; must identify the serving instance         |

## 2. Scope of this phase

In scope: hexagonal skeleton, contract, domain model, use cases, HTTP port, Postgres adapters, wiring.

Explicitly deferred (separate phases, decided later):

- Luhn / expiry **implementation** (only signatures in `domain`).
- Secure PAN storage (encryption / tokenization). Under research by the human.
- Idempotency via `Idempotency-Key` (built; see section 5 and D3).
- AWS Secrets Manager (credentials), CloudWatch (observability).
- Dockerfile, ECS Fargate task definition and service, deployment on AWS.

## 3. Architecture

Hexagonal + DDD Lite + CQRS-lite, from the `go-architecture` skill (*Go with the Domain*).

**Classification:** application layer (`app/command`, `app/query`) is required. Creating a card
stores the idempotency key with the card in one statement, and the read model differs from the write model (masked PAN, `last4` search, cursor pagination).

```text
api/openapi/cards.yml            HTTP contract (source of truth)
cmd/api/main.go                  entry point
scripts/openapi-http.sh          code generation
Makefile                         make openapi_http
internal/
├── domain/                      pure Go, no infrastructure imports
│   ├── card.go                  Card entity (private fields: id, cardHolderName, brand, last4, panHash,
│   │                            expMonth, expYear, createdAt, updatedAt), NewCard, UnmarshalCardFromDatabase,
│   │                            validatePAN (Luhn), detectBrand and validateExpiry -> signatures only, called by NewCard
│   ├── repository.go            CardRepository interface
│   └── errors.go                ErrCardNotFound, ErrInvalidPAN, ErrCardExpired, ...
├── app/
│   ├── app.go                   Application{Commands, Queries}
│   ├── command/create_card.go
│   └── query/{get_card,list_cards,search_cards}.go
├── ports/                       primary adapter (Chi + generated OpenAPI server)
│   ├── http.go                  HttpServer implements generated ServerInterface
│   └── *.gen.go                 generated, never edited
├── adapters/repository/
│   ├── tables/                  row structs, schema, shared errors
│   ├── card_repository.go
│   (idempotency_repository.go deferred)
├── config/                      env loading and validation, read once at startup
└── service/application.go       composition root
```

Dependency rule: `domain` imports nothing internal; `app` -> `domain`; `ports` -> `app`;
`adapters` -> `domain`; only `service` imports everything.

## 4. Decisions

| # | Decision | Rationale |
| :- | :------- | :-------- |
| D1 | Contract-first HTTP with OpenAPI + `oapi-codegen` + Chi | Convention of the architecture skill; contract is reviewable documentation |
| D2 | PostgreSQL (RDS) with GORM (`gorm.io/gorm` + `gorm.io/driver/postgres`) | Human decision. GORM stays inside `adapters/repository`; `gorm` tags live only on the row structs in `tables/`, never on domain entities. The transactional `updateFn` uses `db.Transaction` with `clause.Locking{Strength: "UPDATE"}` |
| D3 | The idempotency key is a unique nullable column on `cards`, in the same Postgres DB | One `INSERT ... ON CONFLICT (idempotency_key) DO NOTHING` is atomic and needs no transaction, table or extra infra. A separate table and Redis stay as documented improvements (requirements C9, C10) |
| D4 | Luhn and expiry validation live in `domain/card.go`, enforced by `NewCard` | Business rules belong to the entity; no entry point can create an invalid card |
| D5 | API, domain and persistence models are separate | Avoid leaking DB/PAN data through the API |
| D6 | The domain never calls `time.Now()`: `NewCardConfig` receives `CreatedAt`, so domain rules stay deterministic in tests. `command.CreateCard` reads the clock directly with `time.Now().UTC()`, with no injected clock | Testability of the rules without extra types (AGENTS.md, Simplicity). Handler tests only check that `createdAt` is close to now |
| D7 | `/health` returns instance identity (`POD_NAME` if set, hostname otherwise) | Lets operators tell replicas apart behind the load balancer; useful behind an ECS service or a load balancer; `POD_NAME` is optional and the hostname (the task or container id on ECS Fargate) is the normal source |
| D8 | Card ids are UUIDv7, generated in `command.CreateCard` with `uuid.NewV7()` from `google/uuid`, not injected; `domain.CardID` stays a string and the column type is `uuid` | Time-ordered ids keep the Postgres primary key index compact and make cursor pagination simple. Handler tests check the id parses as a UUID version 7. An id that is not a valid UUID is reported as not found by the repository adapter |
| D9 | Constructors of app handlers keep panicking on nil dependencies | Human decision for now: failing at startup is intended; revisit returning `(T, error)` later |
| D10 | `Card` also carries `cardHolderName`, `brand`, `panHash` and `updatedAt`. `brand` is derived from the PAN inside the domain; `panHash` is computed outside the domain and received as an opaque string; `updatedAt` equals `createdAt` on creation and is injected like every other timestamp | Human decision. The domain must not know the hashing algorithm or its salt (those belong to secure PAN storage); `panHash` and the PAN never appear in API responses |
| D11 | `panHash = HMAC-SHA256(pan, salt)`, computed in `command.CreateCard` through a `PANHasher` port defined next to the command; the HMAC adapter lives in `adapters/security`. Brand is derived by `detectBrand` in the domain (Visa, Mastercard, Amex, others by prefix), signature only until the Luhn phase | Human decision. Hashing needs a secret, so it is an application concern and the domain only receives the result. The same card always yields the same hash, which enables duplicate detection without storing the PAN. HMAC rather than a plain `SHA256(pan+salt)` to avoid length-extension and misuse |
| D12 | Configuration is loaded once at startup by `internal/config` (environment variables only) and handed to `service`; no other package reads the environment. `APP_ENV` is required (`local` or `cloud`, no default). `local` reads `DB_PASSWORD` and `PAN_SALT` from the environment and never creates the AWS client. `cloud` requires `AWS_REGION`, `DB_PASSWORD_SECRET_ID` and `PAN_SALT_SECRET_ID`, ignores the plain values and reads both secrets as plain-text SecretString from AWS Secrets Manager through `adapters/secrets` (AWS SDK v2). In `cloud` it fails closed: an unreadable or blank secret stops startup before the HTTP port opens. Logging: `LOG_LEVEL` (`debug|info|warn|error`, default `info`) and `LOG_FORMAT` (`text|json`, default `text` in `local` and `json` in `cloud`); an invalid value is reported with the other configuration errors. The logger is built once in `main` with `log/slog` and injected | Human decision (AWS services). Explicit mode instead of implicit detection, so a plain value can never silently replace a secret, and a fast failure on a missing value. The PAN space is small, so the salt must stay secret and its rotation needs a plan |
| D13 | `GET /cards:search` keeps `last4` required and adds optional filters `brand` (exact), `cardHolderName` (case-insensitive prefix) and `expMonth`/`expYear`, combined with AND, plus `limit` and `cursor` with the same rules and cursor as the list | Human decision. `last4` has only 10,000 values, so the result can be large and needs a bound; the extra filters narrow it without turning the endpoint into an unbounded listing. The holder name is personal data, so only a prefix match is offered and it needs an index on `lower(card_holder_name)` |
| D14 | All timestamps are UTC end to end: the connection sets `TimeZone=UTC`, GORM `NowFunc` returns UTC, rows are mapped with `.UTC()` and the API serializes them as RFC 3339 with `Z` | Human requirement. The Postgres driver returns `timestamptz` in the machine zone otherwise, so list and search differed from create |

## 5. Idempotency semantics

- `POST /cards` requires `Idempotency-Key`; missing -> `400 invalid_request`, blank -> `400 idempotency_key_required`.
- First request with a key: the card is inserted with the key in `cards.idempotency_key`.
- Same key again: the insert does nothing and the original card is returned (same `201` and body), no new card.
- Only the key is compared, not the payload: reusing a key with a different card returns the original one (a `422` for that case is a possible improvement).
- Concurrent requests with the same key: the second waits on the unique index and then returns the original card.
- A different key with an already registered PAN still fails with `409` through the unique `pan_hash`.
- Validation errors happen before the insert, so they never consume the key.
- Keys do not expire.

## 6. Open points

- **PAN protection:** what is stored in `cards` (encrypted PAN, token, hash for `last4` search) depends on the pending research. Phase 1 keeps a `PANProtector`-style port undecided and stores only what the research allows.
- **Salt rotation:** changing the salt invalidates every stored `panHash`; decide versioning or re-hashing before production.
- **Cursor format:** the cursor is the id of the last row of the page; pages are ordered by `id` and fetched with `id > cursor`. It relies on UUIDv7 ids (D8) being time-ordered and unique. Trade-off: ordering is by id, not by exact `created_at`.
- **`GET /cards:search`:** literal path, no conflict with `/cards/{id}`; verify with the generator in T04.
