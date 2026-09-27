#!/usr/bin/env node
/**
 * Route-coverage check for api/openapi.yaml.
 *
 * Extracts every controller-decorated route from the NestJS sources under
 * src/ (@Controller prefix + @Get/@Post/@Patch/@Put/@Delete paths) and
 * cross-checks them against the paths declared in api/openapi.yaml, in
 * both directions. Exits 1 on any mismatch.
 *
 * Usage: node contract/route-coverage.mjs
 */
import { readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import yaml from 'js-yaml';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');

const CONTROLLERS = [
  'src/app.controller.ts',
  'src/auth/auth.controller.ts',
  'src/ai/ai.controller.ts',
  'src/categories/categories.controller.ts',
  'src/topics/topics.controller.ts',
  'src/questions/questions.controller.ts',
  'src/sessions/sessions.controller.ts',
];

const METHOD_DECORATORS = ['Get', 'Post', 'Patch', 'Put', 'Delete'];

/** Reads a decorator string argument, e.g. `'live/start'` -> `live/start`. */
function readRouteArg(source, decorator, fromIndex) {
  const at = source.indexOf(`@${decorator}(`, fromIndex);
  if (at === -1) {
    return null;
  }
  const open = at + decorator.length + 2;
  const close = source.indexOf(')', open);
  const arg = source.slice(open, close).trim();
  const match = arg.match(/^['"`](.*?)['"`]$/);
  return { path: match ? match[1] : '', end: close };
}

/** Joins a controller prefix and a route path into a URL path. */
function joinPath(prefix, routePath) {
  const parts = [prefix, routePath].filter((part) => part && part !== '/');
  return `/${parts.join('/')}`;
}

/** Converts a Nest `:param` path into an OpenAPI `{param}` path. */
function toOpenApiPath(nestPath) {
  return nestPath.replace(/:([^/]+)/g, '{$1}');
}

function extractControllerRoutes(file) {
  const source = readFileSync(join(root, file), 'utf8');
  const controllerMatch = source.match(/@Controller\(\s*(?:['"`](.*?)['"`])?\s*\)/);

  if (!controllerMatch) {
    throw new Error(`No @Controller decorator found in ${file}`);
  }

  const prefix = controllerMatch[1] ?? '';
  const routes = [];

  for (const method of METHOD_DECORATORS) {
    let from = 0;
    for (;;) {
      const found = readRouteArg(source, method, from);
      if (!found) {
        break;
      }
      routes.push({
        method: method.toUpperCase(),
        path: toOpenApiPath(joinPath(prefix, found.path)),
        controller: file,
      });
      from = found.end;
    }
  }

  return routes;
}

function extractSpecRoutes(spec) {
  const routes = [];
  for (const [path, item] of Object.entries(spec.paths ?? {})) {
    for (const method of Object.keys(item)) {
      if (['get', 'post', 'patch', 'put', 'delete'].includes(method)) {
        routes.push({ method: method.toUpperCase(), path });
      }
    }
  }
  return routes;
}

const key = (route) => `${route.method} ${route.path}`;

const controllerRoutes = CONTROLLERS.flatMap(extractControllerRoutes);
const spec = yaml.load(readFileSync(join(root, 'api/openapi.yaml'), 'utf8'));
const specRoutes = extractSpecRoutes(spec);

const controllerKeys = new Set(controllerRoutes.map(key));
const specKeys = new Set(specRoutes.map(key));

const missingInSpec = controllerRoutes.filter((route) => !specKeys.has(key(route)));
const extraInSpec = specRoutes.filter((route) => !controllerKeys.has(key(route)));

console.log(`Controller routes: ${controllerRoutes.length}`);
console.log(`Spec routes:       ${specRoutes.length}`);
console.log('');
console.log('Route matrix (controller -> spec):');
for (const route of controllerRoutes) {
  const covered = specKeys.has(key(route)) ? 'OK ' : 'MISSING';
  console.log(`  [${covered}] ${key(route)}  (${route.controller})`);
}

let failed = false;

if (missingInSpec.length > 0) {
  failed = true;
    console.log('\nMissing from api/openapi.yaml:');
  for (const route of missingInSpec) {
    console.log(`  ${key(route)}  (${route.controller})`);
  }
}

if (extraInSpec.length > 0) {
  failed = true;
  console.log('\nDeclared in api/openapi.yaml but not implemented:');
  for (const route of extraInSpec) {
    console.log(`  ${key(route)}`);
  }
}

/** Collects every local $ref in the spec, recursively. */
function collectRefs(node, refs) {
  if (Array.isArray(node)) {
    node.forEach((item) => collectRefs(item, refs));
    return;
  }
  if (node && typeof node === 'object') {
    for (const [key2, value] of Object.entries(node)) {
      if (key2 === '$ref' && typeof value === 'string') {
        refs.push(value);
      } else {
        collectRefs(value, refs);
      }
    }
  }
}

/** Resolves a `#/...` JSON pointer against the spec root. */
function resolvePointer(specRoot, ref) {
  return ref
    .slice(2)
    .split('/')
    .reduce(
      (node, segment) => node?.[segment.replace(/~1/g, '/').replace(/~0/g, '~')],
      specRoot,
    );
}

const structuralProblems = [];

const refs = [];
collectRefs(spec, refs);
for (const ref of [...new Set(refs)]) {
  if (!ref.startsWith('#/')) {
    structuralProblems.push(`non-local $ref: ${ref}`);
  } else if (resolvePointer(spec, ref) === undefined) {
    structuralProblems.push(`unresolvable $ref: ${ref}`);
  }
}

for (const [path, item] of Object.entries(spec.paths ?? {})) {
  const pathParams = [...path.matchAll(/\{([^}]+)\}/g)].map((m) => m[1]);
  for (const [method, operation] of Object.entries(item)) {
    if (!['get', 'post', 'patch', 'put', 'delete'].includes(method)) {
      continue;
    }
    const label = `${method.toUpperCase()} ${path}`;
    const declared = new Set(
      (operation.parameters ?? [])
        .map((param) =>
          param.$ref ? resolvePointer(spec, param.$ref) : param,
        )
        .filter((param) => param?.in === 'path')
        .map((param) => param.name),
    );
    for (const param of pathParams) {
      if (!declared.has(param)) {
        structuralProblems.push(`${label}: path param {${param}} not declared`);
      }
    }
    const responses = operation.responses ?? {};
    if (!Object.keys(responses).some((code) => code.startsWith('2'))) {
      structuralProblems.push(`${label}: no 2xx response declared`);
    }
    if (!operation.operationId) {
      structuralProblems.push(`${label}: missing operationId`);
    }
  }
}

if (structuralProblems.length > 0) {
  failed = true;
  console.log('\nStructural problems:');
  for (const problem of structuralProblems) {
    console.log(`  ${problem}`);
  }
}

if (failed) {
  console.error('\nRoute coverage check FAILED');
  process.exit(1);
}

console.log('\nRoute coverage check PASSED');
