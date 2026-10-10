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
- Idempotency via `Idempotency-Key` (section 5 and D3 are a draft only; nothing is built in this phase).
- AWS Secrets Manager (credentials), CloudWatch (observability).
- Dockerfile, Kubernetes manifests, deployment on AWS.

## 3. Architecture

Hexagonal + DDD Lite + CQRS-lite, from the `go-architecture` skill (*Go with the Domain*).

**Classification:** application layer (`app/command`, `app/query`) is required. Creating a card
will coordinate two repositories (cards + idempotency keys) in one transaction once idempotency is
added, and the read model differs from the write model (masked PAN, `last4` search, cursor pagination).

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
│   ├── repository.go            CardRepository interface (IdempotencyRepository deferred)
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
| D3 | Idempotency stored in the same Postgres DB | Atomic with card insert; no extra infra. Redis stays as a documented alternative |
| D4 | Luhn and expiry validation live in `domain/card.go`, enforced by `NewCard` | Business rules belong to the entity; no entry point can create an invalid card |
| D5 | API, domain and persistence models are separate | Avoid leaking DB/PAN data through the API |
| D6 | The domain never calls `time.Now()`: `NewCardConfig` receives `CreatedAt`, so domain rules stay deterministic in tests. `command.CreateCard` reads the clock directly with `time.Now().UTC()`, with no injected clock | Testability of the rules without extra types (AGENTS.md, Simplicity). Handler tests only check that `createdAt` is close to now |
| D7 | `/health` returns instance identity (`POD_NAME` if set, hostname otherwise) | Lets operators tell replicas apart behind the load balancer; useful behind a k8s Service |
| D8 | Card ids are UUIDv7, generated in `command.CreateCard` with `uuid.NewV7()` from `google/uuid`, not injected; `domain.CardID` stays a string and the column type is `uuid` | Time-ordered ids keep the Postgres primary key index compact and make cursor pagination simple. Handler tests check the id parses as a UUID version 7. An id that is not a valid UUID is reported as not found by the repository adapter |
| D9 | Constructors of app handlers keep panicking on nil dependencies | Human decision for now: failing at startup is intended; revisit returning `(T, error)` later |
| D10 | `Card` also carries `cardHolderName`, `brand`, `panHash` and `updatedAt`. `brand` is derived from the PAN inside the domain; `panHash` is computed outside the domain and received as an opaque string; `updatedAt` equals `createdAt` on creation and is injected like every other timestamp | Human decision. The domain must not know the hashing algorithm or its salt (those belong to secure PAN storage); `panHash` and the PAN never appear in API responses |
| D11 | `panHash = HMAC-SHA256(pan, salt)`, computed in `command.CreateCard` through a `PANHasher` port defined next to the command; the HMAC adapter lives in `adapters/security`. Brand is derived by `detectBrand` in the domain (Visa, Mastercard, Amex, others by prefix), signature only until the Luhn phase | Human decision. Hashing needs a secret, so it is an application concern and the domain only receives the result. The same card always yields the same hash, which enables duplicate detection without storing the PAN. HMAC rather than a plain `SHA256(pan+salt)` to avoid length-extension and misuse |
| D12 | Configuration is loaded once at startup by `internal/config` (environment variables only: port, database settings, AWS region, secret ids, `POD_NAME`) and handed to `service`; no other package reads the environment. The salt and the database credentials come from AWS Secrets Manager through `adapters/secrets` (AWS SDK v2), with an environment fallback for local runs | Human decision (AWS services). One place to see and validate every setting, and a fast failure on a missing value. The PAN space is small, so the salt must stay secret and its rotation needs a plan |
| D13 | `GET /cards:search` keeps `last4` required and adds optional filters `brand` (exact), `cardHolderName` (case-insensitive prefix) and `expMonth`/`expYear`, combined with AND, plus `limit` and `cursor` with the same rules and cursor as the list | Human decision. `last4` has only 10,000 values, so the result can be large and needs a bound; the extra filters narrow it without turning the endpoint into an unbounded listing. The holder name is personal data, so only a prefix match is offered and it needs an index on `lower(card_holder_name)` |

## 5. Idempotency semantics (DEFERRED draft, not part of this phase)

- `POST /cards` requires `Idempotency-Key`; missing -> `400`.
- First request: process, store `(key, request_hash, response)` in the same transaction as the card.
- Same key + same payload hash: return the stored response (same status and body), no new card.
- Same key + different payload hash: `422` (key reuse with a different request).
- Concurrent requests with the same key: serialized by the unique constraint on the key.

## 6. Open points

- **PAN protection:** what is stored in `cards` (encrypted PAN, token, hash for `last4` search) depends on the pending research. Phase 1 keeps a `PANProtector`-style port undecided and stores only what the research allows.
- **Salt rotation:** changing the salt invalidates every stored `panHash`; decide versioning or re-hashing before production.
- **Cursor format:** opaque cursor over `(created_at, id)`; exact encoding in T07.
- **`GET /cards:search`:** literal path, no conflict with `/cards/{id}`; verify with the generator in T04.
