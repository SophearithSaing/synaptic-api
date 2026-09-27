# Contract artifacts (Phase 0)

Behavioral contract of the NestJS API on `main`, captured on
`feat/go-contract` for the Go rewrite. Only `api/` and `contract/` from
this branch merge into `feat/go-rewrite`.

| Path | Contents |
| --- | --- |
| `../api/openapi.yaml` | OpenAPI 3.0 contract for all 33 routes: auth, roles, CSRF, cookies, validation, defaults, error bodies, status codes. |
| `route-coverage.mjs` | Check: every controller-decorated route appears in the spec (plus `$ref`/structural checks). Run: `node contract/route-coverage.mjs`. |
| `routes.mjs` | Shared controller route inventory used by the tooling. |
| `capture/` | Golden-fixture harness: boots NestJS against local MongoDB, Together stubbed, replays scenarios, writes normalized fixtures. See `capture/README.md`. |
| `fixtures/` | Golden request/response JSON per route and case (119 files) plus the generated coverage matrix `index.json`. Regenerate with `node contract/capture/run.mjs` (byte-identical). |
| `audit/` | Read-only BSON audit (`run.mjs`) and the aggregate-counts report (`report.md`) of all 11 collections. |

Reading order for a cold reader: `api/openapi.yaml` for the contract,
`fixtures/index.json` for proven behavior per route, `audit/report.md`
for the legacy data shapes the BSON mappers must tolerate.
