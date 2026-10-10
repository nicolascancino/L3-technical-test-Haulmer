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
| GET    | `/cards:search?last4=`     | Search cards by last 4 digits of the PAN             |
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
│   ├── card.go                  Card entity (private fields), NewCard, UnmarshalCardFromDatabase,
│   │                            validatePAN (Luhn) and validateExpiry -> signatures only, called by NewCard
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
| D6 | Time and IDs injected, never `time.Now()` in domain | Testability |
| D7 | `/health` returns instance identity (`POD_NAME` if set, hostname otherwise) | Required by the challenge; useful behind a k8s Service |
| D8 | Card ids are UUIDv7, generated in `service` with `google/uuid`; `domain.CardID` stays a string and the column type is `uuid` | Time-ordered ids keep the Postgres primary key index compact and make cursor pagination simple. An id that is not a valid UUID is reported as not found by the repository adapter |
| D9 | Constructors of app handlers keep panicking on nil dependencies | Human decision for now: failing at startup is intended; revisit returning `(T, error)` later |

## 5. Idempotency semantics (DEFERRED draft, not part of this phase)

- `POST /cards` requires `Idempotency-Key`; missing -> `400`.
- First request: process, store `(key, request_hash, response)` in the same transaction as the card.
- Same key + same payload hash: return the stored response (same status and body), no new card.
- Same key + different payload hash: `422` (key reuse with a different request).
- Concurrent requests with the same key: serialized by the unique constraint on the key.

## 6. Open points

- **PAN protection:** what is stored in `cards` (encrypted PAN, token, hash for `last4` search) depends on the pending research. Phase 1 keeps a `PANProtector`-style port undecided and stores only what the research allows.
- **Cursor format:** opaque cursor over `(created_at, id)`; exact encoding in T07.
- **`GET /cards:search`:** literal path, no conflict with `/cards/{id}`; verify with the generator in T04.
- **Module path:** `github.com/nicolascancino/L3-technical-test-Haulmer-` (from `origin`).
