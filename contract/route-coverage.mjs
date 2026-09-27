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
import { extractControllerRoutes, routeKey } from './routes.mjs';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');

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

const controllerRoutes = extractControllerRoutes(root);
const spec = yaml.load(readFileSync(join(root, 'api/openapi.yaml'), 'utf8'));
const specRoutes = extractSpecRoutes(spec);

const controllerKeys = new Set(controllerRoutes.map(routeKey));
const specKeys = new Set(specRoutes.map(routeKey));

const missingInSpec = controllerRoutes.filter(
  (route) => !specKeys.has(routeKey(route)),
);
const extraInSpec = specRoutes.filter(
  (route) => !controllerKeys.has(routeKey(route)),
);

console.log(`Controller routes: ${controllerRoutes.length}`);
console.log(`Spec routes:       ${specRoutes.length}`);
console.log('');
console.log('Route matrix (controller -> spec):');
for (const route of controllerRoutes) {
  const covered = specKeys.has(routeKey(route)) ? 'OK ' : 'MISSING';
  console.log(`  [${covered}] ${routeKey(route)}  (${route.controller})`);
}

let failed = false;

if (missingInSpec.length > 0) {
  failed = true;
  console.log('\nMissing from api/openapi.yaml:');
  for (const route of missingInSpec) {
    console.log(`  ${routeKey(route)}  (${route.controller})`);
  }
}

if (extraInSpec.length > 0) {
  failed = true;
  console.log('\nDeclared in api/openapi.yaml but not implemented:');
  for (const route of extraInSpec) {
    console.log(`  ${routeKey(route)}`);
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
      (node, segment) =>
        node?.[segment.replace(/~1/g, '/').replace(/~0/g, '~')],
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
        .map((param) => (param.$ref ? resolvePointer(spec, param.$ref) : param))
        .filter((param) => param?.in === 'path')
        .map((param) => param.name),
    );
    for (const param of pathParams) {
      if (!declared.has(param)) {
        structuralProblems.push(
          `${label}: path param {${param}} not declared`,
        );
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
