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
| Rule | Validate PAN (Luhn) and expiry | Done: 12 to 19 digits, Luhn, expiry valid through the end of its month |
| Rule | Store the PAN securely | Partly: `panHash` (plan D11); encrypted storage still under research |
| Rule | Idempotency by `Idempotency-Key` | Done: unique column on `cards` (plan D3, section 5) |
| Infra | HTTPS load balancer, 2+ replicas in private subnets | Deferred |
| Infra | RDS Postgres or DynamoDB | Postgres chosen (D2) |
| Infra | Secrets Manager for credentials and salt | Designed (D12), built in T09 |
| Infra | Basic logs and metrics in CloudWatch | Deferred |
| Endpoint | `GET /health` with an instance identifier | Done in T06 (D7); wiring in T09 |
| Deliverable | Simple architecture diagram | Done: Mermaid diagram in `README.md` (target design, not deployed) |
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
- Request traceability: a validated `X-Request-Id` (client-provided or generated) is returned in the response header and the error body (`requestId`), and logged with the path, so support can find a request from the id a client reports.

## C. Optional improvements (to review once the project is finished)

Creation and data model:

- **C1** Unique constraint on `panHash` and `409 Conflict` for a duplicate card (done: `ErrCardAlreadyExists` answers `409` `card_already_exists`).
- **C2** Optional: an update operation (`PATCH /cards/{id}`). Until then there is no endpoint and `updatedAt` is an audit field, equal to `createdAt` on creation.
- **C3** Validation of `cardHolderName` (length, allowed characters) and of `last4` as digits.
- **C4** Reject trailing data after the JSON body on `POST /cards`.

Operations:

- **C5** `GET /ready` that checks the database, next to `/health`, so the load balancer only routes to ready replicas.
- **C6** Structured JSON logs with a request id, never logging the PAN or the body.
- **C7** Request metrics (count, latency, status) for CloudWatch.
- **C8** Serve the OpenAPI document from the service.

Idempotency:

- **C9** Store idempotency keys in Redis (ElastiCache) instead of Postgres. Native TTL would expire old keys without a cleanup job, and it fits very high write volume. Trade-off: the key and the card no longer commit in one transaction, so it needs `in progress` and `completed` states and a recovery path for a crash between the two writes. Postgres is kept for the exercise because the key and the card are atomic and no extra infrastructure is needed (plan D3).
- **C11** Reject an `Idempotency-Key` longer than a sane limit (for example 255 characters). The generated server does not enforce `minLength` or `maxLength`, and a very long key only fails when it reaches the unique index.
- **C12** Return `422` when an `Idempotency-Key` is reused with a different card (compare the stored `pan_hash`, or a hash of the whole request) instead of returning the original card.
- **C10** Expire idempotency keys (for example after 24 hours) with a periodic cleanup, or a TTL if C9 is adopted.

Reversible PAN storage:

- **C13** Store the PAN so it can be returned later, only if a flow needs it (charging, showing it to an authorized user). The hash stays for duplicate detection. Options, from simplest to strongest:
  - **Envelope encryption with AWS KMS**: encrypt the PAN with a data key from KMS (`GenerateDataKey`), store the ciphertext and the encrypted data key in a `pan_encrypted` column, and decrypt only through `kms:Decrypt`. IAM and the KMS key policy decide who can decrypt, and CloudTrail audits every use. Rotate the KMS key periodically.
  - **Tokenization**: keep the PAN in a separate, restricted vault (its own database and service) and store only an opaque token in `cards`. The cards service and its logs never see the PAN, which also shrinks the PCI DSS scope. It costs one more service to run.
  - **A payment provider vault** (Stripe, Adyen or similar): the PAN never reaches this system. This is the usual answer when PCI DSS compliance is a goal.

  Rules for any option: add a dedicated endpoint (for example `POST /cards/{id}/reveal`) protected by authentication and authorization, never include the PAN in the normal `Card` response, never log it, and never store the CVV.

  Envelope encryption flow (KMS option): on `POST /cards`, ask KMS for a data key (`GenerateDataKey`), encrypt the PAN with the plaintext key using AES-GCM, store the ciphertext and the encrypted data key, and discard the plaintext key. To read it back, send the encrypted data key to KMS (`Decrypt`), decrypt the PAN with the returned key and discard it. In code this is a `PANCipher` port in the domain with a `KMSPANCipher` adapter. `POST /cards` keeps its contract and only adds the encryption step internally; `GET /cards/{id}` never returns the PAN; recovery goes through the dedicated endpoint, which needs authentication first (the API has none today).
