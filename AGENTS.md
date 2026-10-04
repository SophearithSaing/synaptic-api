# Project: Synaptic

**Goal:** Interactive AI-driven education platform for computing theory.

## Technical Constraints

- **Frontend:** Angular 17+, SCSS (Custom Styles only, NO Tailwind).
- **Backend:** Go (standard `net/http` only, MongoDB driver v2). No router
  or web framework dependencies. The legacy NestJS implementation lives on
  the `main` branch for contract reference only.
- **Database:** MongoDB (single-node replica set locally; transactions are
  required).

## Commands

- `make test` — unit tests (fast, no Docker).
- `make test-integration` — all tests including MongoDB containers.
- `make test-race` — unit tests with the race detector.
- `make vet` / `make fmt` — static analysis and formatting.
- `make ci` — the full gate every branch must pass.
- `make docker-up` — local MongoDB on :27017.

## Code Style

- **Indentation:** 2 spaces (no tabs) in non-Go files; Go uses `gofmt`.
- **Quotes:** Use single quotes `'` for strings unless double quotes are
  required for JSON.
- **Line Width:** Keep code blocks under 80 characters per line where
  possible.
- **Go:** `gofmt` is mandatory (`make fmt-check` runs in CI). In non-test Go
  files, every declared function and every exported identifier MUST have a doc
  comment beginning with its name. No body comments except for complex
  algorithmic logic or non-obvious workarounds for third-party bugs.
