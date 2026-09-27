#!/usr/bin/env node
/**
 * Golden-fixture capture harness.
 *
 * Boots the compiled NestJS app (dist/, run `npm run build` first) against
 * the LOCAL replica-set MongoDB from the worktree .env, resets and seeds
 * deterministic data, replays the scenario list in cases.mjs with the
 * Together API stubbed, and writes normalized request/response fixtures
 * to contract/fixtures/.
 *
 * Usage (from the repository root):
 *
 *   npm run build && node contract/capture/run.mjs
 *
 * Re-running reproduces byte-identical fixtures (see README.md for the
 * normalization rules). Never point DB_URI at Atlas.
 */
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execSync } from 'node:child_process';
import { getConnectionToken } from '@nestjs/mongoose';

import { extractControllerRoutes, routeKey } from '../routes.mjs';
import { createCaptureApp } from './bootstrap.mjs';
import { installTogetherStub, stubState } from './together-stub.mjs';
import { resetAndSeed, SEED_IDS, SEED_PASSWORD } from './seed.mjs';
import { CaptureClient } from './http-client.mjs';
import { Normalizer } from './normalize.mjs';
import { defineCases } from './cases.mjs';

const root = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const fixturesDir = join(root, 'contract', 'fixtures');

const CAPTURED_REQUEST_HEADERS = [
  'content-type',
  'cookie',
  'x-csrf-token',
  'authorization',
];

/** Safety guard: refuse to run against anything but local MongoDB. */
function assertLocalDatabase() {
  const uri = process.env.DB_URI ?? '';

  if (!uri.includes('localhost') && !uri.includes('127.0.0.1')) {
    throw new Error(
      `Refusing to capture fixtures against non-local DB_URI: ${uri}`,
    );
  }
}

/** Builds the fixtures directory name for a route key. */
export function routeDirName(route) {
  if (route === 'GET /') {
    return 'get-root';
  }

  return route
    .toLowerCase()
    .replace(/[{}]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '');
}

async function main() {
  assertLocalDatabase();

  if (!existsSync(join(root, 'dist', 'app.module.js'))) {
    console.log('dist/ not found, running npm run build...');
    execSync('npm run build', { cwd: root, stdio: 'inherit' });
  }

  const realFetch = installTogetherStub();
  const app = await createCaptureApp();
  const connection = app.get(getConnectionToken());
  await resetAndSeed(connection);

  const port = app.getHttpServer().address().port;
  const client = new CaptureClient(`http://127.0.0.1:${port}`, realFetch);
  const normalizer = new Normalizer();

  for (const id of Object.values(SEED_IDS)) {
    normalizer.register('objectId', id);
  }

  const matrix = new Map();
  const state = {};
  let currentKey = 'bootstrap';

  const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

  const prep = (options) =>
    client.request({ ...options, fixtureKey: `${currentKey}:prep` });

  /** Prep request sharing the case's throttler bucket (429 scenarios). */
  const counted = (options) =>
    client.request({ ...options, fixtureKey: currentKey });

  const ensureCsrf = async (jar) => {
    if (!client.jar(jar).get('csrf_token')) {
      const response = await prep({ method: 'GET', path: '/auth/csrf', jar });

      if (response.status !== 200) {
        throw new Error(`ensureCsrf(${jar}) failed: ${response.status}`);
      }
    }
  };

  const authedPrep = async (jar, method, path, body) => {
    await ensureCsrf(jar);

    return prep({
      method,
      path,
      jar,
      body,
      headers: { 'x-csrf-token': client.jar(jar).get('csrf_token') },
    });
  };

  const login = async (jar, identifier) => {
    const response = await authedPrep(jar, 'POST', '/auth/login', {
      identifier,
      password: SEED_PASSWORD,
    });

    if (response.status !== 201) {
      throw new Error(
        `login(${identifier}) failed: ${response.status} ` +
          JSON.stringify(response.body),
      );
    }
  };

  const setAiMode = (mode) => {
    stubState.mode = mode;
  };

  const ctx = {
    state,
    ids: SEED_IDS,
    password: SEED_PASSWORD,
    prep,
    counted,
    ensureCsrf,
    authedPrep,
    login,
    setAiMode,
    sleep,
    client,
  };

  const cap = async (def) => {
    currentKey = `${def.route}#${def.case}`;
    stubState.mode = 'ok';

    try {
      if (def.setup) {
        await def.setup(ctx);
      }

      const dynamicHeaders =
        typeof def.headers === 'function'
          ? def.headers(ctx.state)
          : def.headers;
      const headers = { ...(dynamicHeaders ?? {}) };

      if (def.csrf && def.jar) {
        await ensureCsrf(def.jar);
        headers['x-csrf-token'] = client.jar(def.jar).get('csrf_token');
      }

      const path = typeof def.path === 'function' ? def.path(state) : def.path;
      const body = typeof def.body === 'function' ? def.body(state) : def.body;

      const response = await client.request({
        method: def.method,
        path,
        jar: def.jar,
        body,
        headers,
        fixtureKey: currentKey,
      });

      if (
        def.expectStatus !== undefined &&
        response.status !== def.expectStatus
      ) {
        throw new Error(
          `${currentKey}: expected ${def.expectStatus}, got ` +
            `${response.status}: ${JSON.stringify(response.body)}`,
        );
      }

      const responseHeaders = { ...response.headers };

      if (responseHeaders['retry-after'] !== undefined) {
        responseHeaders['retry-after'] = '<retryAfter>';
      }

      const fixture = {
        route: def.route,
        case: def.case,
        ...(def.note ? { note: def.note } : {}),
        request: {
          method: def.method,
          path,
          headers: Object.fromEntries(
            CAPTURED_REQUEST_HEADERS.flatMap((name) =>
              response.requestHeaders[name]
                ? [[name, response.requestHeaders[name]]]
                : [],
            ),
          ),
          ...(body !== undefined ? { body } : {}),
        },
        response: {
          status: response.status,
          headers: responseHeaders,
          body: response.body,
        },
      };

      const file = join(routeDirName(def.route), `${def.case}.json`);

      if (matrix.has(file)) {
        throw new Error(`duplicate fixture case: ${file}`);
      }

      matrix.set(file, fixture);

      await sleep(2);

      return response;
    } finally {
      stubState.mode = 'ok';
      currentKey = 'teardown';
    }
  };

  ctx.cap = cap;

  await defineCases(ctx);

  // Verify the scenario list covers exactly the controller route inventory.
  const controllerRoutes = extractControllerRoutes(root);
  const inventory = new Set(controllerRoutes.map(routeKey));
  const seenRoutes = new Set(
    [...matrix.values()].map((fixture) => fixture.route),
  );

  const missing = [...inventory].filter((route) => !seenRoutes.has(route));
  const unknown = [...seenRoutes].filter((route) => !inventory.has(route));

  if (missing.length > 0 || unknown.length > 0) {
    for (const route of missing) {
      console.error(`  no fixture case for route: ${route}`);
    }
    for (const route of unknown) {
      console.error(`  fixture case for unknown route: ${route}`);
    }
    throw new Error('fixture matrix does not match the route inventory');
  }

  // Write fixtures (clean rewrite for byte-identical output).
  rmSync(fixturesDir, { recursive: true, force: true });
  mkdirSync(fixturesDir, { recursive: true });

  for (const [file, fixture] of matrix) {
    const target = join(fixturesDir, file);
    const serialized = normalizer.apply(JSON.stringify(fixture, null, 2));
    mkdirSync(dirname(target), { recursive: true });
    writeFileSync(target, `${serialized}\n`);
  }

  const index = {
    routes: [...inventory].sort().map((route) => ({
      route,
      cases: [...matrix.entries()]
        .filter(([, fixture]) => fixture.route === route)
        .map(([file, fixture]) => ({
          case: fixture.case,
          file,
          status: fixture.response.status,
        })),
    })),
  };

  writeFileSync(
    join(fixturesDir, 'index.json'),
    `${JSON.stringify(index, null, 2)}\n`,
  );

  console.log(
    `Captured ${matrix.size} fixture cases across ${inventory.size} routes ` +
      `(Together calls intercepted: ${stubState.calls})`,
  );

  await app.close();
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
