# Synaptic API

Backend API for Synaptic, an interactive AI-driven education platform for
computing theory. Written in Go using only the standard library `net/http`
package for HTTP (no router framework) and the official MongoDB driver v2.
The legacy NestJS implementation lives on the `main` branch for contract
reference.

## Prerequisites

- Go (version pinned in `go.mod`)
- Docker with the Compose plugin (for local MongoDB and integration tests)

## Quick start

```bash
make docker-up              # start local MongoDB (single-node replica set)
cp .env.example .env        # then fill in real values
make migrate                # migrate stored references to ObjectIDs
make run                    # start the API on :3000
```

The replica set is required: the service relies on multi-document
transactions for its idempotency guarantees.

## Commands

| Command                 | Purpose                                          |
| ----------------------- | ------------------------------------------------ |
| `make build`            | Build the API binary to `bin/api`                |
| `make run`              | Run the API with `go run`                        |
| `make migrate`          | Apply required MongoDB schema migrations         |
| `make test`             | Unit tests (no Docker needed)                    |
| `make test-integration` | All tests, incl. MongoDB containers (needs Docker) |
| `make test-race`        | Unit tests with the race detector                |
| `make vet`              | `go vet`                                         |
| `make fmt`              | Format with `gofmt`                              |
| `make ci`               | Full gate: fmt-check, vet, race tests, build     |
| `make docker-up`        | Start local MongoDB on :27017                    |
| `make docker-down`      | Stop local MongoDB                               |
| `make docker-logs`      | Follow MongoDB logs                              |

## Database migrations

Run `make migrate` before starting a newer API version. The catalog migration
renames `topics.category` to `topics.categoryId` and `questionSets.topic` to
`questionSets.topicId`, converting hexadecimal string references to BSON
`ObjectID` values. The migration is transactional and aborts on conflicting or
invalid data.

## Endpoints

- `GET /` — legacy root route (`Hello World!`)
- `GET /health/live` — liveness probe
- `GET /health/ready` — readiness probe (checks MongoDB)

Feature routes are added by the branches in the implementation plan.

## Project layout

```text
cmd/api/            API executable
internal/app/       composition root (wiring)
internal/config/    environment loading and validation
internal/web/       HTTP server, router, middleware, request/response
internal/mongostore/ MongoDB client and BSON persistence
internal/testutil/  integration test helpers (testcontainers)
```

## Configuration

All configuration is environment-based and validated at startup; see
`.env.example`. JWT durations use the legacy compact syntax (`ms`, `s`, `m`,
`h`, `d`). `APP_ENV=production` switches auth cookies to
`Secure` + `SameSite=None`; other environments use `SameSite=Lax` without
`Secure` so plain-HTTP local development works.
