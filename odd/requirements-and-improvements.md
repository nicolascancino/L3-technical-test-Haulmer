# Requirements and improvements (cards-api)

Tracks what the original brief asked for, what we decided to improve on our own, and the ideas still to explore.
The plan ([`plans/cards-api.md`](plans/cards-api.md)) holds the decisions; this file only classifies them.

## A. Original requirements

| Area | Requirement | Status |
| :--- | :---------- | :----- |
| Endpoint | `POST /cards` creates a card | Contract, app, HTTP port done; persistence pending (T07-T09) |
| Endpoint | `GET /cards/{id}` | Same |
| Endpoint | `GET /cards?limit=&cursor=` simple pagination | Same |
| Endpoint | `GET /cards:search?last4=` | Same |
| Rule | Validate PAN (Luhn) and expiry | Deferred: signatures only in `domain` |
| Rule | Store the PAN securely | Partly: `panHash` (plan D11); encrypted storage still under research |
| Rule | Idempotency by `Idempotency-Key` | Deferred: draft in plan section 5 |
| Infra | HTTPS load balancer, 2+ replicas in private subnets | Deferred |
| Infra | RDS Postgres or DynamoDB | Postgres chosen (D2) |
| Infra | Secrets Manager for credentials and salt | Designed (D12), built in T09 |
| Infra | Basic logs and metrics in CloudWatch | Deferred |
| Endpoint | `GET /health` with an instance identifier | Done in T06 (D7); wiring in T09 |
| Deliverable | Simple architecture diagram | Pending |
| Deliverable | Evidence of load distribution | Pending |

## B. Improvements already adopted

- Contract-first HTTP with OpenAPI and generated server (D1); the contract is the reviewable documentation.
- Hexagonal layers with a CQRS-lite split, and separate API, domain and persistence models (D5).
- Time-ordered UUIDv7 ids (D8).
- Error semantics: `400` malformed, `422` business rule, `404` unknown, `413` body over 1 MiB, fixed message on `500`.
- Responses never expose the PAN or the hash; only `last4`, brand, holder, expiry and timestamps.
- Extra card attributes: `cardHolderName`, `brand`, `panHash`, `updatedAt` (D10).
- `panHash = HMAC-SHA256(pan, salt)` computed in the application layer (D11).
- Single configuration entry point loaded at startup (D12).
- Search keeps `last4` required, adds optional `brand`, `cardHolderName` (prefix), `expMonth`/`expYear` filters combined with AND, and `limit`/`cursor` pagination (D13).
- The CVV is not requested or stored (it cannot be persisted after authorization and no rule needs it).

## C. Optional improvements (to review once the project is finished)

Creation and data model:

- **C1** Unique constraint on `panHash` and `409 Conflict` for a duplicate card (today `ErrCardAlreadyExists` is not mapped).
- **C2** Optional: an update operation (`PATCH /cards/{id}`). Until then there is no endpoint and `updatedAt` is an audit field, equal to `createdAt` on creation.
- **C3** Validation of `cardHolderName` (length, allowed characters) and of `last4` as digits.
- **C4** Reject trailing data after the JSON body on `POST /cards`.

Operations:

- **C5** `GET /ready` that checks the database, next to `/health`, so the load balancer only routes to ready replicas.
- **C6** Structured JSON logs with a request id, never logging the PAN or the body.
- **C7** Request metrics (count, latency, status) for CloudWatch.
- **C8** Serve the OpenAPI document from the service.
