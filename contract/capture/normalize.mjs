/**
 * Volatile-value normalizer for golden fixtures.
 *
 * Applied to the serialized fixture JSON before writing.
 *
 * Identity values whose equality is timing-independent are replaced by
 * stable `<type:NN>` tokens numbered in order of first appearance across
 * the whole capture run (random per issuance, so value collisions cannot
 * happen and numbering is byte-stable):
 *
 * 1. refreshToken — `<24-hex>.<43-char base64url>` refresh cookie values
 * 2. csrfToken    — bare 43-char base64url CSRF tokens
 * 3. objectId     — 24-char hex Mongo ObjectIds (seed ids pre-registered)
 *
 * Everything else volatile is replaced by CONSTANT tokens with no value
 * tracking:
 *
 * 4. `<jwt>`      — compact JWTs (`eyJ…`.…`.…`) in bodies, headers,
 *                   cookies. JWTs for the same user issued within the
 *                   same second are byte-identical (`iat` has second
 *                   granularity), so numbering them breaks re-runs.
 * 5. `<isoDate>`  — `YYYY-MM-DDTHH:mm:ss.sssZ` timestamps
 * 6. `<httpDate>` — `Expires=Wed, 01 Jan 2026 00:00:00 GMT` cookie dates
 *
 * Value-equality between timestamps is wall-clock dependent (same-second
 * cookie Expires collisions, millisecond-boundary flips between
 * `startedAt` and `createdAt`), so numbering them breaks byte-identical
 * re-runs. Constant tokens are immune. Values that are equal by
 * construction (e.g. `submittedAt`/`evaluatedAt`) still read as equal.
 *
 * `Retry-After` header values are replaced structurally by the capturer
 * (`<retryAfter>`), and `X-RateLimit-*`/`Date`/`ETag`/`Content-Length`
 * response headers are never captured (see README).
 */
/**
 * Rules apply in list order; earlier matches win. `type` rules replace
 * each distinct value with a numbered `<type:NN>` token; `token` rules
 * replace every match with the same constant token.
 */
const RULES = [
  // JWT first: its 43-char segments must not be eaten by csrfToken.
  { token: '<jwt>', pattern: /eyJ[\w-]*\.[\w-]*\.[\w-]*/g },
  // Refresh token before its 24-hex session id / 43-char secret parts.
  { type: 'refreshToken', pattern: /[0-9a-f]{24}\.[\w-]{43}/g },
  { type: 'csrfToken', pattern: /(?<![\w-])[\w-]{43}(?![\w-])/g },
  { type: 'objectId', pattern: /\b[0-9a-fA-F]{24}\b/g },
  {
    token: '<isoDate>',
    pattern: /\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z/g,
  },
  {
    token: '<httpDate>',
    pattern: /\w{3}, \d{2} \w{3} \d{4} \d{2}:\d{2}:\d{2} GMT/g,
  },
];

export class Normalizer {
  constructor() {
    /** @type {Map<string, Map<string, string>>} type -> raw -> token */
    this.maps = new Map();
  }

  /** Pre-registers a raw value so it receives a stable early token. */
  register(type, rawValue) {
    this.tokenFor(type, rawValue);
  }

  /** Returns (creating when needed) the token for a raw value. */
  tokenFor(type, rawValue) {
    if (!this.maps.has(type)) {
      this.maps.set(type, new Map());
    }

    const map = this.maps.get(type);

    if (!map.has(rawValue)) {
      const token = `<${type}:${String(map.size + 1).padStart(2, '0')}>`;
      map.set(rawValue, token);
    }

    return map.get(rawValue);
  }

  /** Replaces every volatile value in the given text with its token. */
  apply(text) {
    let result = text;

    for (const rule of RULES) {
      if (rule.type) {
        result = result.replace(rule.pattern, (match) =>
          this.tokenFor(rule.type, match),
        );
      } else {
        result = result.replace(rule.pattern, rule.token);
      }
    }

    return result;
  }
}
