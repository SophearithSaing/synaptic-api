/**
 * Shared route extraction for the contract tooling.
 *
 * Reads the NestJS controllers under src/ and returns every
 * controller-decorated route as `{ method, path, controller }` with the
 * path in OpenAPI form (`:id` converted to `{id}`).
 */
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

export const CONTROLLERS = [
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

/** Extracts all decorated routes from one controller file. */
export function extractControllerFileRoutes(root, file) {
  const source = readFileSync(join(root, file), 'utf8');
  const controllerMatch = source.match(
    /@Controller\(\s*(?:['"`](.*?)['"`])?\s*\)/,
  );

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

/** Extracts all decorated routes from every controller. */
export function extractControllerRoutes(root) {
  return CONTROLLERS.flatMap((file) => extractControllerFileRoutes(root, file));
}

/** Formats a route as `METHOD /path`. */
export function routeKey(route) {
  return `${route.method} ${route.path}`;
}
