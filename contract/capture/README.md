# Fixture capture harness

Boots the compiled NestJS app against the **local** replica-set MongoDB,
resets and seeds deterministic data, replays a fixed scenario list with
the Together API stubbed, and writes golden request/response fixtures to
`contract/fixtures/`.

## Usage

```sh
make docker-up          # from the Go tree: local MongoDB on :27017 (rs0)
npm ci && npm run build # in this worktree
node contract/capture/run.mjs
```

Prerequisites:

- Worktree `.env`: copy of the Go tree's `.env` with `DB_URI` overridden
  to `mongodb://localhost:27017/?replicaSet=rs0` (gitignored, never
  committed). The harness **refuses to run against a non-local DB_URI**.
- `dist/` build output (`npm run build`; the harness builds once when
  `dist/app.module.js` is missing — rebuild manually after src changes).

## Determinism check

```sh
node contract/capture/run.mjs
cp -r contract/fixtures /tmp/fixtures-run1
node contract/capture/run.mjs
diff -r /tmp/fixtures-run1 contract/fixtures   # must print nothing
```

Every run drops and re-seeds the database, replays scenarios in a fixed
order, and rewrites `contract/fixtures/` from scratch, so output is
byte-identical across runs.

## How it works

- `run.mjs` — orchestrator: safety guard, boot, seed, capture, matrix
  assertion (every controller-decorated route must have at least one
  case, checked against `contract/routes.mjs`), fixture writing.
- `bootstrap.mjs` — mirrors `src/main.ts` (cookie parser, 5 MB body
  limits, CORS, global ValidationPipe) on an ephemeral port.
- `together-stub.mjs` — replaces `globalThis.fetch` before boot; the
  `together-ai` SDK captures the global fetch at client construction, so
  all calls to `*.together.ai` are answered deterministically in-process
  (no real provider calls, no secrets in fixtures). Modes: `ok`
  (schema-valid question/evaluations), `invalid-json`, `empty`,
  `http-error`; switched per scenario by the harness.
- `seed.mjs` — fixed ObjectIds (`5eed…`), fixed timestamps, two users
  (`student`/`admin`, password `Password1`), one category, two topics
  (one without question sets), level-0 MCQ and level-1 written question
  sets. Indexes are rebuilt after the drop so unique-index behavior
  (register 409, duplicate-slug 500) is genuine.
- `http-client.mjs` — cookie-jar client (Path-aware, clear-cookie
  handling) capturing status, selected headers, and body.
- `normalize.mjs` — volatile-value normalizer (rules below).
- `cases.mjs` — the scenario list (119 cases across the 33 routes).

## Harness-only accommodations

- **Throttler buckets**: `ThrottlerGuard.getTracker` is patched (process
  level, app code untouched) to key buckets by the `x-fixture-key`
  request header instead of the client IP. Each case uses its own key so
  unrelated cases cannot interfere (all harness traffic shares one IP);
  the 429 scenarios deliberately share one key across their setup
  requests and hit the real limits (register 3/min, login 5/min).
- **AI stub sleeps 5 ms per call** so `createdAt`-sorted collections
  (aiLogs) have deterministic ordering at MongoDB's millisecond
  resolution; a short sleep also separates session creation timestamps.
- `x-fixture-key` itself is harness plumbing and is excluded from the
  captured request headers.

## Normalization rules

Applied to the serialized fixture JSON (`contract/capture/normalize.mjs`):

| Token               | Replaced value                                            |
| ------------------- | --------------------------------------------------------- |
| `<refreshToken:NN>` | `<24-hex>.<base64url-43>` refresh cookie values           |
| `<csrfToken:NN>`    | bare 43-char base64url CSRF tokens                        |
| `<objectId:NN>`     | 24-char hex ObjectIds (seed ids are pre-registered)       |
| `<jwt>`             | compact JWTs (Authorization header, cookies)              |
| `<isoDate>`         | `YYYY-MM-DDTHH:mm:ss.sssZ` timestamps                     |
| `<httpDate>`        | `Expires=…GMT` cookie dates                               |
| `<retryAfter>`      | `Retry-After` header values (replaced structurally)       |

`NN` tokens are numbered in order of first appearance across the run;
their underlying values are random per issuance, so numbering is stable.
JWTs and timestamps get **constant** tokens because their value-equality
is wall-clock dependent (JWT `iat` has second granularity; same-second
cookie `Expires` collisions; millisecond flips between `startedAt` and
`createdAt`) and numbering them broke byte-identical re-runs. Values
equal by construction (e.g. `submittedAt`/`evaluatedAt`) still read as
equal.

## Header capture policy

Captured response headers: `content-type`, `set-cookie` (normalized),
`retry-after` (normalized). Deliberately not captured:

- `x-ratelimit-limit/-remaining/-reset` — `Reset` is wall-clock derived
- `date`, `etag`, `content-length`, `connection`, `keep-alive`,
  `x-powered-by` — transport noise

Captured request headers: `content-type`, `cookie` (normalized),
`x-csrf-token` (normalized), `authorization` (normalized).

## Fixture layout

`contract/fixtures/<route-dir>/<case>.json` — one file per scenario
(`routeDirName` in run.mjs maps `DELETE /sessions/live/{id}` to
`delete-sessions-live-id`). `contract/fixtures/index.json` is the
generated coverage matrix: every route with its cases, files, and
statuses.

## Secrets and PII

Seed users are synthetic, the AI provider is stubbed, and `.env` is
gitignored. No real credentials, provider output, or user data appear in
fixtures.
