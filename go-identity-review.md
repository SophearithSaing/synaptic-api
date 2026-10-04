# Go Identity Branch Review

Base branch: `feat/go-rewrite`

Reviewed branch: `feat/go-identity`

Validation completed:

- `make ci`
- `git diff --check`

`make test-integration` requires Docker and could not run after the fixes because
the Docker socket is not accessible in this environment.

## Findings

### 1. Router aliases are not named types

- Commit: `f26d08b`
- File: `internal/web/router.go`
- Category: Suggestion
- Status: RESOLVED

```go
type Middleware = func(http.Handler) http.Handler
type MountFunc = func(mux *http.ServeMux)
```

These declarations are aliases, so they do not introduce type identity. Use
defined function types instead:

```go
type Middleware func(http.Handler) http.Handler
type MountFunc func(*http.ServeMux)
```

Method values remain assignable to these types.

### 2. Throttler mixes injected and real time

- Commit: `f2d490b`
- File: `internal/web/throttle.go`
- Category: Concern
- Status: RESOLVED

`hit` accepts `now`, but blocked-window calculations use `time.Until`, which
reads the real clock. Pruning also uses a hard-coded hour unrelated to the
configured TTL.

```go
if now.Before(window.blockedUntil) {
  return false, time.Until(window.blockedUntil)
}

if window.epoch.Add(time.Hour).Before(now) {
  delete(th.buckets, key)
}
```

Use `window.blockedUntil.Sub(now)` and store the actual configured window
expiry instead of applying a separate cleanup duration.

### 3. Rate limiting is tied to one process and transport peer

- Commit: `f2d490b`
- File: `internal/web/throttle.go`
- Category: Safety
- Status: RESOLVED

Resolved after the parent authorization round: the throttled window
state now persists in a shared store. `mongostore.ThrottleStore`
implements `web.ThrottleStore` over the `throttles` collection with a
version-based compare-and-set loop per key (no unsafe read-write
pair), receiving the request context for every operation. Startup
bootstraps a TTL index on the purge marker (the later of the window
horizon and an active block) so stale buckets expire across
replicas. Client identity resolves through an explicitly trusted-proxy
aware `web.ForwardedForClientIP` config from the validated
`THROTTLE_TRUSTED_PROXIES` list (exact addresses or CIDRs): forwarded
headers count only while the transport peer and every hop between the
client and the process are trusted; otherwise requests key on the
transport peer, so spoofed headers cannot rotate buckets, and with no
trusted proxies configured the direct-peer behavior is preserved. The
in-memory store remains for unit tests and locally isolated
constructions.

The limiter stores buckets in process memory and keys them using
`r.RemoteAddr`. Limits reset on restart, can be bypassed across replicas, and
group every client behind a reverse proxy into the same bucket.

Use a trusted client-IP resolver and shared rate-limit storage, or explicitly
constrain deployment to one directly exposed process.

### 4. `Chain` is a pass-through wrapper

- Commit: `26036ef`
- File: `internal/web/middleware.go`
- Category: Suggestion
- Status: RESOLVED

```go
func Chain(
  handler http.Handler,
  middleware ...func(http.Handler) http.Handler,
) http.Handler {
  return chain(handler, middleware...)
}
```

Rename the existing `chain` implementation to `Chain` and use it internally
instead of maintaining an exported wrapper around a private function.

### 5. Authenticator depends on an oversized interface

- Commit: `26036ef`
- File: `internal/identity/middleware.go`
- Category: Concern
- Status: RESOLVED

`Authenticator` only calls `FindUserByID`, but it accepts the complete
`Repository` interface. Catalog tests consequently implement eight unrelated
methods with `panic("not used")`.

Define a consumer-side interface containing only `FindUserByID`. This follows
the Go convention of accepting the smallest interface needed by the caller.

### 6. `Service.UserByID` is an unused pass-through

- Commit: `26036ef`
- File: `internal/identity/service.go`
- Category: Suggestion
- Status: RESOLVED

```go
func (s *Service) UserByID(
  ctx context.Context,
  id string,
) (*User, error) {
  return s.repo.FindUserByID(ctx, id)
}
```

No production code calls this method; the authenticator accesses the
repository directly. Remove it or deliberately make the authenticator depend
on the service.

### 7. Authentication validation is stringly typed

- Commit: `26036ef`
- Files: `internal/identity/handler.go`, `internal/identity/validate.go`
- Category: Concern
- Status: RESOLVED

Validation is implemented as a mini framework composed of:

- `map[string]json.RawMessage`
- `map[string]fieldRules`
- separate field-order slices
- a separate transform map
- a `map[string]string` result
- a second decode of each field after validation

These structures must remain manually synchronized. Prefer typed register and
login request values with explicit transformation and `Validate` methods.

### 8. Custom validation differs from the legacy contract

- Commit: `26036ef`
- File: `internal/identity/validate.go`
- Category: Safety
- Status: RESOLVED

Length validation uses `len`, which counts UTF-8 bytes rather than characters.
A short password containing multibyte runes can therefore satisfy the
eight-character minimum. The simplified email regular expression also accepts
addresses rejected by the legacy `IsEmail` validator.

Use Unicode-aware character counting and an email validator matching the
required contract.

### 9. JWT claims use an untyped map and manual extraction

- Commit: `26036ef`
- File: `internal/identity/token.go`
- Category: Suggestion
- Status: RESOLVED

`jwt.MapClaims`, `claimString`, and `VerifiedToken` manually recreate a typed
claims model through runtime assertions.

Define a claims struct embedding `jwt.RegisteredClaims`, add `Email` and
`Username`, and use `ParseWithClaims`.

### 10. Login performs two user queries

- Commit: `26036ef`
- Files: `internal/identity/service.go`, `internal/mongostore/identity.go`
- Category: Concern
- Status: RESOLVED

Login first loads the user and then loads the password hash by user ID. This is
an unnecessary second query and allows the two reads to observe different
states.

Return an authentication-specific record containing both the user and password
hash from one repository query.

### 11. Repository failures are converted to authentication failures

- Commit: `26036ef`
- Files: `internal/identity/service.go`,
  `internal/identity/middleware.go`
- Category: Concern
- Status: RESOLVED

Several paths combine `err != nil` with a missing result and return
`ErrUnauthorized`. MongoDB outages are consequently reported as 401 responses.

Handle repository errors separately and propagate them. Return unauthorized
only for successful lookups that do not find a valid identity or session.

### 12. Registration is not transactional

- Commit: `26036ef`
- File: `internal/identity/service.go`
- Category: Concern
- Status: RESOLVED

User creation and refresh-session creation are separate writes. If session
creation fails, the account remains without credentials and a retry conflicts
with its username or email.

Create the user and initial session in one MongoDB transaction.

### 13. Logout discards revocation failures

- Commit: `26036ef`
- File: `internal/identity/handler.go`
- Category: Safety
- Status: RESOLVED

```go
_ = h.service.Logout(r.Context(), cookieValue(cookie))
```

The handler reports successful logout and clears browser cookies even when the
server-side refresh session could not be revoked. Propagate the error while
still applying an appropriate cookie-clearing policy.

### 14. Inserted ObjectIDs are converted through `any`

- Commit: `26036ef`
- File: `internal/mongostore/identity.go`
- Category: Concern
- Status: RESOLVED

```go
func objectIDHex(value any) string {
  if objectID, ok := value.(bson.ObjectID); ok {
    return objectID.Hex()
  }
  return ""
}
```

An unexpected inserted ID becomes a successful empty ID. Assign a
`bson.NewObjectID()` before insertion and return that known value, or return an
error when the assertion fails.

### 15. Duplicate-key handling parses rendered error text

- Commit: `26036ef`
- File: `internal/mongostore/identity.go`
- Category: Concern
- Status: RESOLVED

```go
message := err.Error()
switch {
case strings.Contains(message, "username_1"):
case strings.Contains(message, "email_1"):
}
```

This depends on MongoDB's rendered message and specific index names. Inspect
`mongo.WriteException` and its raw key pattern/details through `errors.As`
instead.

### 16. Identity indexes exist only in test code

- Commit: `6b8c125`
- File: `internal/testutil/identity.go`
- Category: Concern
- Status: RESOLVED

Unique username/email indexes and the auth-session TTL index are created by a
test helper, but there is no production migration or startup equivalent. A
fresh production or local database therefore has different behavior from the
integration tests.

Move index management into a production migration or explicit startup
bootstrap.

### 17. ObjectID validation duplicates the MongoDB driver

- Commit: `cf3310f`
- Files: `internal/catalog/handler.go`, `internal/mongostore/catalog.go`
- Category: Suggestion
- Status: RESOLVED

```go
var objectIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{24}$`)

func validObjectID(id string) bool {
  return objectIDPattern.MatchString(id)
}
```

The handler validates with a custom regular expression, after which the
repository parses the same value with `bson.ObjectIDFromHex`. Parse once, or
have the repository return a distinct invalid-ID error that maps to HTTP 400.

### 18. Catalog error handling is implemented through presentation types

- Commit: `cf3310f`
- Files: `internal/catalog/model.go`, `internal/catalog/handler.go`
- Category: Concern
- Status: RESOLVED

Catalog errors contain client-facing messages and are recognized through
concrete type assertions. Wrapped errors no longer match, while every future
`catalog.Error` will be treated as a 404.

Use separate sentinel errors with `errors.Is`, then map each sentinel to a
fixed HTTP message in the handler.

### 19. Catalog response helpers hide repository semantics

- Commit: `cf3310f`
- File: `internal/catalog/handler.go`
- Category: Suggestion
- Status: RESOLVED

`notFound` is named like a predicate but writes an HTTP response and returns
flow-control state. `catalogMessage` repeats its type assertion. `orEmpty`
patches nil repository slices at every HTTP call site.

Use a direct error switch in handlers and initialize repository list results as
non-nil empty slices.

### 20. `QuestionSet.Topic` removes type safety

- Commit: `cf3310f`
- File: `internal/catalog/model.go`
- Category: Concern
- Status: RESOLVED

```go
Topic any `json:"topic"`
```

The field represents a string-or-object union using unrestricted `any`. Define
a dedicated topic value type or consistently carry the serialized form as
`json.RawMessage`.

### 21. `ToRawJSON` has an invalid variadic contract

- Commit: `cf3310f`
- File: `internal/mongostore/catalog_raw.go`
- Category: Concern
- Status: RESOLVED

The function is exported and variadic, but only one value produces valid JSON.
Zero values produce an empty payload and multiple values are concatenated
without separators.

Make it an unexported function accepting exactly one `bson.RawValue`.

### 22. Catalog uses a partial custom BSON-to-JSON encoder

- Commit: `cf3310f`
- File: `internal/mongostore/catalog_raw.go`
- Category: Concern
- Status: RESOLVED

The custom encoder rejects valid BSON types including binary, decimal, regex,
and timestamps. Because question documents are intentionally loose, one such
value can unexpectedly turn an otherwise valid catalog response into a 500.

Use a defined response model or a complete, tested conversion boundary rather
than a partial streaming encoder.

### 23. Topic conversion performs a lossy JSON round trip

- Commit: `cf3310f`
- File: `internal/mongostore/catalog.go`
- Category: Concern
- Status: RESOLVED

`topicHexOrState` converts BSON to JSON, decodes that JSON into `any`, and then
the response encoder serializes it again. Errors silently become `nil`, and
large integers can lose precision through `float64`.

Return `json.RawMessage` directly instead of decoding it into `any`.

### 24. Question topic handling contains a dead error branch

- Commit: `cf3310f`
- File: `internal/mongostore/catalog.go`
- Category: Suggestion
- Status: RESOLVED

`questionSetTopic` checks whether `topicReference` returned
`mongo.ErrNoDocuments`, but `findRaw` already converts that condition into a
nil result and `topicReference` returns the original reference.

Remove the unreachable special case and return the actual conversion error.

### 25. Catalog population performs N+1 queries

- Commit: `cf3310f`
- File: `internal/mongostore/catalog.go`
- Category: Concern
- Status: RESOLVED

Every topic performs another category lookup. More notably,
`QuestionSetsByTopicSlug` already loads the topic but loads that same topic
again for every question set when `populateTopic=true`.

Reuse the topic document already loaded by slug and batch or cache category
references.

## Uncommitted Changes

No source-code concerns or suggestions were found before this report was
created.

## Implementation Notes

All findings were addressed on `feat/go-identity`. The non-validation changes
are committed in `493b6a5`, `020839d`, `658e309`, `8b3ae7d`, and `982a6f1`.
Findings 7 and 8 now use typed request values and
`go-playground/validator`; the old schema, order, transform maps, and duplicate
field decoding are removed in `6fb8323`. Decisions worth review attention:

- Finding 3 (RESOLVED): window state persists in the shared
  `mongostore.ThrottleStore` (version-based compare-and-set per key,
  context-aware operations, TTL bootstrap on the purge marker), and
  client identity goes through the trusted-proxy-aware
  `web.ForwardedForClientIP` fed from the validated
  `THROTTLE_TRUSTED_PROXIES` list; untrusted peers never see their
  header values honored. The memory store stays for unit tests and
  locally isolated constructions.
- Finding 13: a failed logout now writes the error (generic 500) and
  keeps the auth cookies so the client can retry; success still clears
  cookies with the pinned bare 201. Malformed/missing tokens and secret
  mismatches remain non-failures per the existing contract.
- Finding 11: repository failures now propagate (500s); token
  verification failures and lookup misses stay 401s.
- Finding 15: duplicate-key mapping reads the raw server write error
  (`mongo.WriteException.WriteErrors[].Raw`) and maps the collided
  keyPattern field through `errors.As`; no rendered-text parsing. Needs
  the real-Mongo integration tests to confirm the server document
  carries `keyPattern` for these 11000 errors.
- Finding 22: types without a pinned legacy shape (binary, decimal,
  regex, timestamp, code, DB pointer) render as MongoDB extended JSON
  through the driver, so valid loose documents can no longer fail a
  response; pinned types keep their exact legacy bytes (unit tested).
- Finding 10: login resolves the user and its stored password hash in
  one authentication query (`FindAuthRecordByUsername/Email` returning
  an `identity.AuthRecord`); the separate user lookup and
  `PasswordHash` reads are gone from repository, store, and fakes.
- Finding 19: handlers branch with explicit `if err != nil { ...;
  return }`; `writeLookupError` only maps the error to its pinned
  body and no longer masquerades as a predicate or returns flow
  state.
- Integration tests cannot run in this environment (Docker unavailable,
  permission denied). Unit tests (`make test`), `make vet`, `gofmt`,
  and `make ci` all pass.
