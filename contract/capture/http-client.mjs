/**
 * Minimal cookie-jar HTTP client for fixture capture.
 *
 * Implements just enough RFC 6265 behavior for this API: cookie storage
 * keyed by name, Path-attribute request matching, and removal on
 * clear-cookie (empty value / Max-Age=0 / past Expires).
 */
export class CookieJar {
  constructor() {
    /** @type {Map<string, { value: string, path: string }>} */
    this.cookies = new Map();
  }

  /** Stores cookies from a response's set-cookie header list. */
  store(setCookies) {
    for (const setCookie of setCookies) {
      const [pair, ...attributes] = setCookie.split(';').map((p) => p.trim());
      const separator = pair.indexOf('=');
      const name = pair.slice(0, separator);
      const value = pair.slice(separator + 1);
      const attrs = Object.fromEntries(
        attributes.map((attribute) => {
          const at = attribute.indexOf('=');
          return at === -1
            ? [attribute.toLowerCase(), true]
            : [attribute.slice(0, at).toLowerCase(), attribute.slice(at + 1)];
        }),
      );

      const expired =
        value === '' ||
        attrs['max-age'] === '0' ||
        (typeof attrs.expires === 'string' &&
          Date.parse(attrs.expires) <= Date.now());

      if (expired) {
        this.cookies.delete(name);
      } else {
        this.cookies.set(name, {
          value,
          path: typeof attrs.path === 'string' ? attrs.path : '/',
        });
      }
    }
  }

  /** Returns the Cookie header value for a request path. */
  header(requestPath) {
    const pairs = [];

    for (const [name, cookie] of this.cookies) {
      if (pathMatches(requestPath, cookie.path)) {
        pairs.push(`${name}=${cookie.value}`);
      }
    }

    return pairs.length > 0 ? pairs.join('; ') : undefined;
  }

  /** Reads a single cookie value by name. */
  get(name) {
    return this.cookies.get(name)?.value;
  }
}

/** RFC 6265 section 5.1.4 path-match (simplified for this API). */
function pathMatches(requestPath, cookiePath) {
  if (cookiePath === '/' || requestPath === cookiePath) {
    return true;
  }

  return (
    requestPath.startsWith(cookiePath) &&
    (cookiePath.endsWith('/') || requestPath[cookiePath.length] === '/')
  );
}

/** Headers captured in fixtures (others are transport noise). */
const CAPTURED_RESPONSE_HEADERS = ['content-type', 'set-cookie', 'retry-after'];

/**
 * HTTP client bound to a base URL with per-name cookie jars. Every
 * request carries an `x-fixture-key` header used by the harness to
 * isolate throttler buckets (see README).
 */
export class CaptureClient {
  constructor(baseUrl, fetchImpl) {
    this.baseUrl = baseUrl;
    this.fetchImpl = fetchImpl;
    this.jars = new Map();
  }

  /** Returns (creating when needed) the named cookie jar. */
  jar(name) {
    if (!this.jars.has(name)) {
      this.jars.set(name, new CookieJar());
    }

    return this.jars.get(name);
  }

  /**
   * Performs one request and returns the raw response plus parsed body.
   *
   * @param {object} options
   * @param {string} options.method HTTP method.
   * @param {string} options.path Request path (with query string).
   * @param {string} options.fixtureKey Throttler bucket key header.
   * @param {string} [options.jar] Cookie jar name.
   * @param {object} [options.body] JSON body.
   * @param {object} [options.headers] Extra headers.
   */
  async request({ method, path, fixtureKey, jar, body, headers = {} }) {
    const requestHeaders = {
      'x-fixture-key': fixtureKey,
      ...headers,
    };

    if (jar) {
      const cookieHeader = this.jar(jar).header(path);

      if (cookieHeader) {
        requestHeaders.cookie = cookieHeader;
      }
    }

    let serializedBody;
    if (body !== undefined) {
      requestHeaders['content-type'] = 'application/json';
      serializedBody = JSON.stringify(body);
    }

    const response = await this.fetchImpl(`${this.baseUrl}${path}`, {
      method,
      headers: requestHeaders,
      body: serializedBody,
      redirect: 'manual',
    });

    if (jar) {
      this.jar(jar).store(response.headers.getSetCookie());
    }

    const text = await response.text();
    const contentType = response.headers.get('content-type') ?? '';

    return {
      status: response.status,
      headers: Object.fromEntries(
        CAPTURED_RESPONSE_HEADERS.flatMap((name) => {
          if (name === 'set-cookie') {
            const values = response.headers.getSetCookie();
            return values.length > 0 ? [[name, values]] : [];
          }

          const value = response.headers.get(name);
          return value === null ? [] : [[name, value]];
        }),
      ),
      body: contentType.includes('json') && text ? JSON.parse(text) : text,
      requestHeaders,
    };
  }
}
