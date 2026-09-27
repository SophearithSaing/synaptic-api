/**
 * Fixture scenario list, executed sequentially by run.mjs.
 *
 * Conventions:
 * - `cap(def)` captures one request/response pair as a fixture file.
 * - `ctx.prep(...)` performs uncaptured setup requests (own throttler
 *   bucket), `ctx.counted(...)` performs uncaptured requests in the
 *   case's bucket (only used by 429 scenarios).
 * - `ctx.authedPrep(jar, method, path, body)` attaches the jar's CSRF
 *   pair; `ctx.login(jar, identifier)` logs a seeded user into a jar.
 * - Captured responses are returned, so flows store runtime ids in
 *   `ctx.state` for later cases.
 * - `ctx.setAiMode(mode)` switches the Together stub; it resets to `ok`
 *   after every case.
 *
 * Every route in the controller inventory must appear at least once;
 * run.mjs fails the run otherwise.
 */
export async function defineCases(ctx) {
  const { cap, state, ids, login, authedPrep, counted, setAiMode } = ctx;

  // ------------------------------------------------------------ root
  await cap({
    route: 'GET /',
    case: 'success',
    method: 'GET',
    path: '/',
    expectStatus: 200,
    note: 'Public liveness greeting; no cookies or CSRF required.',
  });

  // ------------------------------------------------------------- csrf
  await cap({
    route: 'GET /auth/csrf',
    case: 'success',
    method: 'GET',
    path: '/auth/csrf',
    jar: 'register',
    expectStatus: 200,
    note: 'Issues the JS-readable csrf_token cookie and body token.',
  });

  // --------------------------------------------------------- register
  await cap({
    route: 'POST /auth/register',
    case: 'success',
    method: 'POST',
    path: '/auth/register',
    jar: 'register',
    csrf: true,
    body: {
      username: 'alice',
      email: 'alice@example.com',
      password: 'Password1',
    },
    expectStatus: 201,
  });

  await cap({
    route: 'POST /auth/register',
    case: 'validation-400',
    method: 'POST',
    path: '/auth/register',
    jar: 'register',
    csrf: true,
    body: {
      username: 'bobby',
      email: 'bobby@example.com',
      password: 'password',
    },
    expectStatus: 400,
    note: 'Password lacks an uppercase letter and a digit.',
  });

  await cap({
    route: 'POST /auth/register',
    case: 'unknown-field-400',
    method: 'POST',
    path: '/auth/register',
    jar: 'register',
    csrf: true,
    body: {
      username: 'carol',
      email: 'carol@example.com',
      password: 'Password1',
      role: 'admin',
    },
    expectStatus: 400,
    note: 'forbidNonWhitelisted rejects the unknown `role` property.',
  });

  await cap({
    route: 'POST /auth/register',
    case: 'conflict-409',
    method: 'POST',
    path: '/auth/register',
    jar: 'register',
    csrf: true,
    body: {
      username: 'student',
      email: 'unique@example.com',
      password: 'Password1',
    },
    expectStatus: 409,
    note: 'Seeded username; uniqueness is case-insensitive.',
  });

  await cap({
    route: 'POST /auth/register',
    case: 'conflict-email-409',
    method: 'POST',
    path: '/auth/register',
    jar: 'register',
    csrf: true,
    body: {
      username: 'uniqueuser',
      email: 'admin@example.com',
      password: 'Password1',
    },
    expectStatus: 409,
  });

  await cap({
    route: 'POST /auth/register',
    case: 'csrf-403',
    method: 'POST',
    path: '/auth/register',
    jar: 'register',
    body: {
      username: 'dave',
      email: 'dave@example.com',
      password: 'Password1',
    },
    expectStatus: 403,
    note: 'csrf_token cookie is sent but the x-csrf-token header is not.',
  });

  await cap({
    route: 'POST /auth/register',
    case: 'throttled-429',
    method: 'POST',
    path: '/auth/register',
    jar: 'register-throttle',
    csrf: true,
    body: {
      username: 'throttle4',
      email: 'throttle4@example.com',
      password: 'Password1',
    },
    expectStatus: 429,
    note: '4th register request within 60s (route limit 3, 5-min block).',
    setup: async () => {
      await ctx.ensureCsrf('register-throttle');
      const token = ctx.client.jar('register-throttle').get('csrf_token');

      for (let n = 1; n <= 3; n += 1) {
        const response = await counted({
          method: 'POST',
          path: '/auth/register',
          jar: 'register-throttle',
          headers: { 'x-csrf-token': token },
          body: {
            username: `throttle${n}`,
            email: `throttle${n}@example.com`,
            password: 'Password1',
          },
        });

        if (response.status !== 201) {
          throw new Error(`throttle setup ${n}: ${response.status}`);
        }
      }
    },
  });

  // ------------------------------------------------------------ login
  await cap({
    route: 'POST /auth/login',
    case: 'success',
    method: 'POST',
    path: '/auth/login',
    jar: 'login',
    csrf: true,
    body: { identifier: 'student', password: ctx.password },
    expectStatus: 201,
  });

  await cap({
    route: 'POST /auth/login',
    case: 'validation-400',
    method: 'POST',
    path: '/auth/login',
    jar: 'login',
    csrf: true,
    body: { identifier: 'bad identifier', password: ctx.password },
    expectStatus: 400,
    note: 'identifier must not contain whitespace.',
  });

  await cap({
    route: 'POST /auth/login',
    case: 'wrong-password-401',
    method: 'POST',
    path: '/auth/login',
    jar: 'login',
    csrf: true,
    body: { identifier: 'student', password: 'WrongPass1' },
    expectStatus: 401,
  });

  await cap({
    route: 'POST /auth/login',
    case: 'throttled-429',
    method: 'POST',
    path: '/auth/login',
    jar: 'login-throttle',
    csrf: true,
    body: { identifier: 'student', password: ctx.password },
    expectStatus: 429,
    note: '6th login request within 60s (route limit 5, 5-min block).',
    setup: async () => {
      await ctx.ensureCsrf('login-throttle');
      const token = ctx.client.jar('login-throttle').get('csrf_token');

      for (let n = 1; n <= 5; n += 1) {
        const response = await counted({
          method: 'POST',
          path: '/auth/login',
          jar: 'login-throttle',
          headers: { 'x-csrf-token': token },
          body: { identifier: 'student', password: ctx.password },
        });

        if (response.status !== 201) {
          throw new Error(`login throttle setup ${n}: ${response.status}`);
        }
      }
    },
  });

  // ------------------------------------------------- me / refresh / logout
  await cap({
    route: 'GET /auth/me',
    case: 'success',
    method: 'GET',
    path: '/auth/me',
    jar: 'student',
    expectStatus: 200,
    setup: () => login('student', 'student'),
  });

  await cap({
    route: 'GET /auth/me',
    case: 'success-bearer',
    method: 'GET',
    path: '/auth/me',
    expectStatus: 200,
    note: 'JWT supplied via Authorization header instead of the cookie.',
    setup: async () => {
      await login('bearer', 'student');
      state.bearerToken = ctx.client.jar('bearer').get('access_token');
    },
    headers: (s) => ({ authorization: `Bearer ${s.bearerToken}` }),
  });

  await cap({
    route: 'GET /auth/me',
    case: 'unauth-401',
    method: 'GET',
    path: '/auth/me',
    jar: 'fresh',
    expectStatus: 401,
  });

  await cap({
    route: 'POST /auth/refresh',
    case: 'success',
    method: 'POST',
    path: '/auth/refresh',
    jar: 'student',
    csrf: true,
    expectStatus: 201,
    note: 'Rotates the refresh token and re-sets both auth cookies.',
  });

  await cap({
    route: 'POST /auth/refresh',
    case: 'unauth-401',
    method: 'POST',
    path: '/auth/refresh',
    jar: 'anon2',
    csrf: true,
    expectStatus: 401,
    note: 'No refresh_token cookie present.',
  });

  await cap({
    route: 'POST /auth/refresh',
    case: 'csrf-403',
    method: 'POST',
    path: '/auth/refresh',
    jar: 'student',
    expectStatus: 403,
  });

  await cap({
    route: 'POST /auth/logout',
    case: 'success',
    method: 'POST',
    path: '/auth/logout',
    jar: 'student2',
    csrf: true,
    expectStatus: 201,
    setup: () => login('student2', 'student'),
    note: 'Revokes the refresh session and clears both auth cookies.',
  });

  await cap({
    route: 'POST /auth/logout',
    case: 'no-cookie-201',
    method: 'POST',
    path: '/auth/logout',
    jar: 'anon3',
    csrf: true,
    expectStatus: 201,
    note: 'No refresh cookie: logout still succeeds and clears cookies.',
  });

  // -------------------------------------------------------- categories
  const createCategoryBody = {
    title: 'Networking',
    slug: 'networking',
    description: 'Networks and protocols.',
    icon: 'wifi',
  };

  const createdCategory = await cap({
    route: 'POST /categories/category/create',
    case: 'success',
    method: 'POST',
    path: '/categories/category/create',
    jar: 'admin',
    csrf: true,
    body: createCategoryBody,
    expectStatus: 201,
    setup: () => login('admin', 'admin'),
  });
  state.createdCategoryId = createdCategory.body.id;

  await cap({
    route: 'POST /categories/category/create',
    case: 'validation-400',
    method: 'POST',
    path: '/categories/category/create',
    jar: 'admin',
    csrf: true,
    body: { title: 'No Icon', slug: 'no-icon', description: 'Missing.' },
    expectStatus: 400,
  });

  await cap({
    route: 'POST /categories/category/create',
    case: 'role-403',
    method: 'POST',
    path: '/categories/category/create',
    jar: 'student',
    csrf: true,
    body: {
      title: 'Forbidden',
      slug: 'forbidden',
      description: 'Student role.',
      icon: 'x',
    },
    expectStatus: 403,
  });

  await cap({
    route: 'POST /categories/category/create',
    case: 'unauth-401',
    method: 'POST',
    path: '/categories/category/create',
    jar: 'anon4',
    csrf: true,
    body: createCategoryBody,
    expectStatus: 401,
    note: 'Valid CSRF pair but no JWT.',
  });

  await cap({
    route: 'POST /categories/category/create',
    case: 'duplicate-slug-500',
    method: 'POST',
    path: '/categories/category/create',
    jar: 'admin',
    csrf: true,
    body: {
      title: 'Core Again',
      slug: 'core-concepts',
      description: 'Duplicate slug.',
      icon: 'cpu',
    },
    expectStatus: 500,
    note: 'Unique-index violation surfaces as 500, not 409.',
  });

  await cap({
    route: 'POST /categories/category/create',
    case: 'csrf-403',
    method: 'POST',
    path: '/categories/category/create',
    jar: 'admin',
    body: createCategoryBody,
    expectStatus: 403,
  });

  await cap({
    route: 'GET /categories/categories',
    case: 'success',
    method: 'GET',
    path: '/categories/categories',
    jar: 'student',
    expectStatus: 200,
  });

  await cap({
    route: 'GET /categories/categories',
    case: 'unauth-401',
    method: 'GET',
    path: '/categories/categories',
    jar: 'fresh',
    expectStatus: 401,
  });

  await cap({
    route: 'GET /categories/{id}',
    case: 'success',
    method: 'GET',
    path: `/categories/${ids.category}`,
    jar: 'student',
    expectStatus: 200,
  });

  await cap({
    route: 'GET /categories/{id}',
    case: 'not-found-404',
    method: 'GET',
    path: `/categories/${ids.unknown}`,
    jar: 'student',
    expectStatus: 404,
  });

  await cap({
    route: 'GET /categories/{id}',
    case: 'invalid-id-400',
    method: 'GET',
    path: '/categories/not-an-id',
    jar: 'student',
    expectStatus: 400,
  });

  await cap({
    route: 'DELETE /categories/{id}',
    case: 'success',
    method: 'DELETE',
    path: (s) => `/categories/${s.createdCategoryId}`,
    jar: 'admin',
    csrf: true,
    expectStatus: 204,
  });

  await cap({
    route: 'DELETE /categories/{id}',
    case: 'not-found-404',
    method: 'DELETE',
    path: `/categories/${ids.unknown}`,
    jar: 'admin',
    csrf: true,
    expectStatus: 404,
  });

  await cap({
    route: 'DELETE /categories/{id}',
    case: 'role-403',
    method: 'DELETE',
    path: `/categories/${ids.category}`,
    jar: 'student',
    csrf: true,
    expectStatus: 403,
  });

  await cap({
    route: 'DELETE /categories/{id}',
    case: 'invalid-id-400',
    method: 'DELETE',
    path: '/categories/not-an-id',
    jar: 'admin',
    csrf: true,
    expectStatus: 400,
  });

  // ------------------------------------------------------------ topics
  const createdTopic = await cap({
    route: 'POST /topics/create',
    case: 'success',
    method: 'POST',
    path: '/topics/create',
    jar: 'admin',
    csrf: true,
    body: {
      title: 'Logic Gates',
      slug: 'logic-gates',
      description: 'Boolean logic and gates.',
      icon: 'gate',
      tags: ['logic'],
      category: ids.category,
    },
    expectStatus: 201,
  });
  state.createdTopicId = createdTopic.body.id;

  await cap({
    route: 'POST /topics/create',
    case: 'validation-400',
    method: 'POST',
    path: '/topics/create',
    jar: 'admin',
    csrf: true,
    body: {
      title: 'Too Many Tags',
      slug: 'too-many-tags',
      description: 'Three tags.',
      icon: 'tag',
      tags: ['a', 'b', 'c'],
      category: ids.category,
    },
    expectStatus: 400,
  });

  await cap({
    route: 'POST /topics/create',
    case: 'role-403',
    method: 'POST',
    path: '/topics/create',
    jar: 'student',
    csrf: true,
    body: {
      title: 'Forbidden',
      slug: 'forbidden-topic',
      description: 'Student role.',
      icon: 'x',
      tags: ['x'],
      category: ids.category,
    },
    expectStatus: 403,
  });

  await cap({
    route: 'POST /topics/create',
    case: 'duplicate-slug-500',
    method: 'POST',
    path: '/topics/create',
    jar: 'admin',
    csrf: true,
    body: {
      title: 'Binary Again',
      slug: 'binary-basics',
      description: 'Duplicate slug.',
      icon: 'binary',
      tags: ['binary'],
      category: ids.category,
    },
    expectStatus: 500,
    note: 'Unique-index violation surfaces as 500, not 409.',
  });

  await cap({
    route: 'GET /topics',
    case: 'success',
    method: 'GET',
    path: '/topics',
    jar: 'student',
    expectStatus: 200,
  });

  await cap({
    route: 'GET /topics',
    case: 'unauth-401',
    method: 'GET',
    path: '/topics',
    jar: 'fresh',
    expectStatus: 401,
  });

  await cap({
    route: 'GET /topics/{id}',
    case: 'success',
    method: 'GET',
    path: `/topics/${ids.topic}`,
    jar: 'student',
    expectStatus: 200,
  });

  await cap({
    route: 'GET /topics/{id}',
    case: 'not-found-404',
    method: 'GET',
    path: `/topics/${ids.unknown}`,
    jar: 'student',
    expectStatus: 404,
  });

  await cap({
    route: 'GET /topics/{id}',
    case: 'invalid-id-400',
    method: 'GET',
    path: '/topics/not-an-id',
    jar: 'student',
    expectStatus: 400,
  });

  await cap({
    route: 'DELETE /topics/{id}',
    case: 'success',
    method: 'DELETE',
    path: (s) => `/topics/${s.createdTopicId}`,
    jar: 'admin',
    csrf: true,
    expectStatus: 204,
  });

  await cap({
    route: 'DELETE /topics/{id}',
    case: 'not-found-404',
    method: 'DELETE',
    path: `/topics/${ids.unknown}`,
    jar: 'admin',
    csrf: true,
    expectStatus: 404,
  });

  await cap({
    route: 'DELETE /topics/{id}',
    case: 'role-403',
    method: 'DELETE',
    path: `/topics/${ids.topic}`,
    jar: 'student',
    csrf: true,
    expectStatus: 403,
  });

  // ------------------------------------------------------- question sets
  const stubMcq = {
    id: 'created-q1',
    type: 'mcq',
    prompt: 'What does CPU stand for?',
    options: [
      { id: 'o1', text: 'Central Processing Unit' },
      { id: 'o2', text: 'Computer Personal Unit' },
      { id: 'o3', text: 'Central Program Utility' },
    ],
    correctOptionId: 'o1',
    targetConcepts: ['hardware-basics'],
    feedback: {
      correct: 'Correct.',
      incorrect: 'Incorrect.',
    },
    rubrics: {
      keyPoints: ['Central processing.'],
      misconceptions: ['Personal computer.'],
    },
  };

  const createdSet = await cap({
    route: 'POST /questions/create',
    case: 'success',
    method: 'POST',
    path: '/questions/create',
    jar: 'admin',
    csrf: true,
    body: [
      {
        topic: ids.topic,
        setType: 'regular',
        level: 2,
        questions: [stubMcq],
      },
    ],
    expectStatus: 201,
    note: 'Body is an array validated item-by-item (ParseArrayPipe).',
  });
  state.createdSetId = createdSet.body[0].id;

  await cap({
    route: 'POST /questions/create',
    case: 'validation-400',
    method: 'POST',
    path: '/questions/create',
    jar: 'admin',
    csrf: true,
    body: [{ topic: ids.topic, setType: 'regular', level: 3 }],
    expectStatus: 400,
  });

  await cap({
    route: 'POST /questions/create',
    case: 'non-array-400',
    method: 'POST',
    path: '/questions/create',
    jar: 'admin',
    csrf: true,
    body: { topic: ids.topic, setType: 'regular', level: 3, questions: [] },
    expectStatus: 400,
    note: 'ParseArrayPipe rejects a non-array body.',
  });

  await cap({
    route: 'POST /questions/create',
    case: 'role-403',
    method: 'POST',
    path: '/questions/create',
    jar: 'student',
    csrf: true,
    body: [
      { topic: ids.topic, setType: 'regular', level: 3, questions: [stubMcq] },
    ],
    expectStatus: 403,
  });

  await cap({
    route: 'PATCH /questions/update',
    case: 'success',
    method: 'PATCH',
    path: '/questions/update',
    jar: 'admin',
    csrf: true,
    body: (s) => [{ id: s.createdSetId, level: 3 }],
    expectStatus: 200,
  });

  await cap({
    route: 'PATCH /questions/update',
    case: 'not-found-404',
    method: 'PATCH',
    path: '/questions/update',
    jar: 'admin',
    csrf: true,
    body: [{ id: ids.unknown, level: 3 }],
    expectStatus: 404,
  });

  await cap({
    route: 'PATCH /questions/update',
    case: 'role-403',
    method: 'PATCH',
    path: '/questions/update',
    jar: 'student',
    csrf: true,
    body: (s) => [{ id: s.createdSetId, level: 3 }],
    expectStatus: 403,
  });

  await cap({
    route: 'PATCH /questions/{id}',
    case: 'success',
    method: 'PATCH',
    path: (s) => `/questions/${s.createdSetId}`,
    jar: 'admin',
    csrf: true,
    body: { level: 4 },
    expectStatus: 200,
  });

  await cap({
    route: 'PATCH /questions/{id}',
    case: 'not-found-404',
    method: 'PATCH',
    path: `/questions/${ids.unknown}`,
    jar: 'admin',
    csrf: true,
    body: { level: 4 },
    expectStatus: 404,
  });

  await cap({
    route: 'PATCH /questions/{id}',
    case: 'invalid-id-400',
    method: 'PATCH',
    path: '/questions/not-an-id',
    jar: 'admin',
    csrf: true,
    body: { level: 4 },
    expectStatus: 400,
  });

  await cap({
    route: 'PATCH /questions/{id}',
    case: 'role-403',
    method: 'PATCH',
    path: (s) => `/questions/${s.createdSetId}`,
    jar: 'student',
    csrf: true,
    body: { level: 4 },
    expectStatus: 403,
  });

  await cap({
    route: 'GET /questions/topic/{slug}',
    case: 'success',
    method: 'GET',
    path: '/questions/topic/binary-basics',
    jar: 'student',
    expectStatus: 200,
    note: 'Topic NOT populated by default on this route.',
  });

  await cap({
    route: 'GET /questions/topic/{slug}',
    case: 'success-populated',
    method: 'GET',
    path: '/questions/topic/binary-basics?populateTopic=true',
    jar: 'student',
    expectStatus: 200,
  });

  await cap({
    route: 'GET /questions/topic/{slug}',
    case: 'not-found-404',
    method: 'GET',
    path: '/questions/topic/no-such-topic',
    jar: 'student',
    expectStatus: 404,
  });

  await cap({
    route: 'GET /questions/{id}',
    case: 'success',
    method: 'GET',
    path: `/questions/${ids.questionSetL0}`,
    jar: 'student',
    expectStatus: 200,
    note: 'Topic IS populated by default on this route (raw document).',
  });

  await cap({
    route: 'GET /questions/{id}',
    case: 'success-unpopulated',
    method: 'GET',
    path: `/questions/${ids.questionSetL0}?populateTopic=false`,
    jar: 'student',
    expectStatus: 200,
  });

  await cap({
    route: 'GET /questions/{id}',
    case: 'not-found-404',
    method: 'GET',
    path: `/questions/${ids.unknown}`,
    jar: 'student',
    expectStatus: 404,
  });

  await cap({
    route: 'GET /questions/{id}',
    case: 'invalid-id-400',
    method: 'GET',
    path: '/questions/not-an-id',
    jar: 'student',
    expectStatus: 400,
  });

  await cap({
    route: 'DELETE /questions/{id}',
    case: 'success',
    method: 'DELETE',
    path: (s) => `/questions/${s.createdSetId}`,
    jar: 'admin',
    csrf: true,
    expectStatus: 204,
  });

  await cap({
    route: 'DELETE /questions/{id}',
    case: 'not-found-404',
    method: 'DELETE',
    path: `/questions/${ids.unknown}`,
    jar: 'admin',
    csrf: true,
    expectStatus: 404,
  });

  await cap({
    route: 'DELETE /questions/{id}',
    case: 'role-403',
    method: 'DELETE',
    path: `/questions/${ids.questionSetL0}`,
    jar: 'student',
    csrf: true,
    expectStatus: 403,
  });

  // ---------------------------------------------------- regular sessions
  const started = await cap({
    route: 'POST /sessions/start',
    case: 'success',
    method: 'POST',
    path: '/sessions/start',
    jar: 'student',
    csrf: true,
    body: { topicId: ids.topic },
    expectStatus: 201,
  });
  state.sessionId = started.body.sessionId;

  await cap({
    route: 'POST /sessions/start',
    case: 'not-found-404',
    method: 'POST',
    path: '/sessions/start',
    jar: 'student',
    csrf: true,
    body: { topicId: ids.unknown },
    expectStatus: 404,
  });

  await cap({
    route: 'POST /sessions/start',
    case: 'no-question-set-404',
    method: 'POST',
    path: '/sessions/start',
    jar: 'student',
    csrf: true,
    body: { topicId: ids.topicEmpty },
    expectStatus: 404,
    note: 'Topic exists but has no level-0 question set.',
  });

  await cap({
    route: 'POST /sessions/start',
    case: 'validation-400',
    method: 'POST',
    path: '/sessions/start',
    jar: 'student',
    csrf: true,
    body: { topicId: 'not-an-id' },
    expectStatus: 400,
  });

  await cap({
    route: 'POST /sessions/start',
    case: 'unauth-401',
    method: 'POST',
    path: '/sessions/start',
    jar: 'anon5',
    csrf: true,
    body: { topicId: ids.topic },
    expectStatus: 401,
  });

  await cap({
    route: 'POST /sessions/start',
    case: 'csrf-403',
    method: 'POST',
    path: '/sessions/start',
    jar: 'student',
    body: { topicId: ids.topic },
    expectStatus: 403,
  });

  await cap({
    route: 'POST /sessions/continue',
    case: 'success',
    method: 'POST',
    path: '/sessions/continue',
    jar: 'student',
    csrf: true,
    body: (s) => ({ sessionId: s.sessionId }),
    expectStatus: 201,
  });

  await cap({
    route: 'POST /sessions/continue',
    case: 'not-found-404',
    method: 'POST',
    path: '/sessions/continue',
    jar: 'student',
    csrf: true,
    body: { sessionId: ids.unknown },
    expectStatus: 404,
  });

  await cap({
    route: 'POST /sessions/continue',
    case: 'validation-400',
    method: 'POST',
    path: '/sessions/continue',
    jar: 'student',
    csrf: true,
    body: { sessionId: 'not-an-id' },
    expectStatus: 400,
  });

  await cap({
    route: 'POST /sessions/submit-answer',
    case: 'success-pass',
    method: 'POST',
    path: '/sessions/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.sessionId,
      questionSetId: ids.questionSetL0,
      answers: [
        { questionId: 'seed-l0-q1', answer: 'o1' },
        { questionId: 'seed-l0-q2', answer: 'o2' },
      ],
    }),
    expectStatus: 201,
    note: 'All MCQ answers correct: session advances to level 1.',
  });

  const started2 = await authedPrep('student', 'POST', '/sessions/start', {
    topicId: ids.topic,
  });
  state.sessionId2 = started2.body.sessionId;
  await ctx.sleep(5);

  await cap({
    route: 'POST /sessions/submit-answer',
    case: 'fail-attempt',
    method: 'POST',
    path: '/sessions/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.sessionId2,
      questionSetId: ids.questionSetL0,
      answers: [
        { questionId: 'seed-l0-q1', answer: 'o2' },
        { questionId: 'seed-l0-q2', answer: 'o1' },
      ],
    }),
    expectStatus: 201,
    note: 'All answers wrong: failed attempt, nextQuestionSet null.',
  });

  await cap({
    route: 'POST /sessions/submit-answer',
    case: 'success-written-pass',
    method: 'POST',
    path: '/sessions/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.sessionId,
      questionSetId: ids.questionSetL1,
      answers: [
        {
          questionId: 'seed-l1-q1',
          answer: 'Invert every bit and add one.',
        },
      ],
    }),
    expectStatus: 201,
    note: 'Written answer evaluated by the (stubbed) AI provider.',
  });

  await cap({
    route: 'POST /sessions/submit-answer',
    case: 'unknown-question-400',
    method: 'POST',
    path: '/sessions/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.sessionId,
      questionSetId: ids.questionSetL0,
      answers: [{ questionId: 'no-such-question', answer: 'x' }],
    }),
    expectStatus: 400,
  });

  await cap({
    route: 'POST /sessions/submit-answer',
    case: 'session-not-found-404',
    method: 'POST',
    path: '/sessions/submit-answer',
    jar: 'student',
    csrf: true,
    body: {
      sessionId: ids.unknown,
      questionSetId: ids.questionSetL0,
      answers: [{ questionId: 'seed-l0-q1', answer: 'o1' }],
    },
    expectStatus: 404,
  });

  await cap({
    route: 'POST /sessions/submit-answer',
    case: 'set-not-found-404',
    method: 'POST',
    path: '/sessions/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.sessionId,
      questionSetId: ids.unknown,
      answers: [{ questionId: 'seed-l0-q1', answer: 'o1' }],
    }),
    expectStatus: 404,
  });

  await cap({
    route: 'POST /sessions/submit-answer',
    case: 'validation-400',
    method: 'POST',
    path: '/sessions/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.sessionId,
      questionSetId: ids.questionSetL0,
      answers: [],
    }),
    expectStatus: 400,
  });

  const started3 = await authedPrep('student', 'POST', '/sessions/start', {
    topicId: ids.topic,
  });
  state.sessionId3 = started3.body.sessionId;
  await ctx.sleep(5);

  await cap({
    route: 'POST /sessions/submit-answer',
    case: 'ai-invalid-503',
    method: 'POST',
    path: '/sessions/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.sessionId3,
      questionSetId: ids.questionSetL1,
      answers: [{ questionId: 'seed-l1-q1', answer: 'Some written answer.' }],
    }),
    expectStatus: 503,
    note: 'Stubbed provider returns non-JSON for the written evaluation.',
    setup: () => setAiMode('invalid-json'),
  });

  await cap({
    route: 'GET /sessions/in-progress',
    case: 'success',
    method: 'GET',
    path: '/sessions/in-progress',
    jar: 'student',
    expectStatus: 200,
    note: 'Active sessions sorted by updatedAt descending.',
  });

  await cap({
    route: 'GET /sessions/in-progress',
    case: 'unauth-401',
    method: 'GET',
    path: '/sessions/in-progress',
    jar: 'fresh',
    expectStatus: 401,
  });

  await cap({
    route: 'DELETE /sessions/{id}',
    case: 'success',
    method: 'DELETE',
    path: (s) => `/sessions/${s.sessionId3}`,
    jar: 'student',
    csrf: true,
    expectStatus: 204,
  });

  await cap({
    route: 'DELETE /sessions/{id}',
    case: 'not-found-404',
    method: 'DELETE',
    path: `/sessions/${ids.unknown}`,
    jar: 'student',
    csrf: true,
    expectStatus: 404,
  });

  await cap({
    route: 'DELETE /sessions/{id}',
    case: 'invalid-id-400',
    method: 'DELETE',
    path: '/sessions/not-an-id',
    jar: 'student',
    csrf: true,
    expectStatus: 400,
  });

  // -------------------------------------------------------- live sessions
  const liveStarted = await cap({
    route: 'POST /sessions/live/start',
    case: 'success',
    method: 'POST',
    path: '/sessions/live/start',
    jar: 'student',
    csrf: true,
    body: { topicId: ids.topic },
    expectStatus: 201,
    note: 'First question generated by the (stubbed) AI provider.',
  });
  state.liveSessionId = liveStarted.body.sessionId;
  state.liveQuestionId = liveStarted.body.questionId;

  await cap({
    route: 'POST /sessions/live/continue',
    case: 'success',
    method: 'POST',
    path: '/sessions/live/continue',
    jar: 'student',
    csrf: true,
    body: (s) => ({ sessionId: s.liveSessionId }),
    expectStatus: 201,
    note: 'Returns the pending question without a new AI call.',
  });

  await cap({
    route: 'POST /sessions/live/continue',
    case: 'not-found-404',
    method: 'POST',
    path: '/sessions/live/continue',
    jar: 'student',
    csrf: true,
    body: { sessionId: ids.unknown },
    expectStatus: 404,
  });

  await cap({
    route: 'POST /sessions/live/continue',
    case: 'validation-400',
    method: 'POST',
    path: '/sessions/live/continue',
    jar: 'student',
    csrf: true,
    body: { sessionId: 'not-an-id' },
    expectStatus: 400,
  });

  const rejected = await cap({
    route: 'POST /sessions/live/reject',
    case: 'success',
    method: 'POST',
    path: '/sessions/live/reject',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.liveSessionId,
      questionId: s.liveQuestionId,
      reason: 'Question was too easy.',
    }),
    expectStatus: 201,
    note: 'Rejection reason is fed back into the generation prompt.',
  });
  state.liveQuestionId = rejected.body.questionId;

  await cap({
    route: 'POST /sessions/live/reject',
    case: 'session-not-found-404',
    method: 'POST',
    path: '/sessions/live/reject',
    jar: 'student',
    csrf: true,
    body: {
      sessionId: ids.unknown,
      questionId: ids.unknown,
      reason: 'No session.',
    },
    expectStatus: 404,
  });

  await cap({
    route: 'POST /sessions/live/reject',
    case: 'question-not-found-404',
    method: 'POST',
    path: '/sessions/live/reject',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.liveSessionId,
      questionId: ids.unknown,
      reason: 'No question.',
    }),
    expectStatus: 404,
  });

  const correct1 = await cap({
    route: 'POST /sessions/live/submit-answer',
    case: 'success-correct',
    method: 'POST',
    path: '/sessions/live/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.liveSessionId,
      questionId: s.liveQuestionId,
      answer: 'binary-basics-l0-q1-o2',
    }),
    expectStatus: 201,
    note: 'Correct MCQ option; the next question is generated.',
  });
  state.liveQuestion2Id = correct1.body.nextQuestion.questionId;

  await cap({
    route: 'POST /sessions/live/submit-answer',
    case: 'fail-answer',
    method: 'POST',
    path: '/sessions/live/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.liveSessionId,
      questionId: s.liveQuestion2Id,
      answer: 'binary-basics-l0-q2-o1',
    }),
    expectStatus: 201,
    note: 'Wrong option before 3 accepted questions: nextQuestion null.',
  });

  await cap({
    route: 'POST /sessions/live/submit-answer',
    case: 'level-complete',
    method: 'POST',
    path: '/sessions/live/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.liveSessionId,
      questionId: s.liveQuestion3Id,
      answer: 'binary-basics-l0-q3-o2',
    }),
    expectStatus: 201,
    note: 'Third accepted question: set attempt persisted, level advances.',
    setup: async () => {
      const retry = await authedPrep(
        'student',
        'POST',
        '/sessions/live/submit-answer',
        {
          sessionId: state.liveSessionId,
          questionId: state.liveQuestion2Id,
          answer: 'binary-basics-l0-q2-o2',
        },
      );

      if (retry.status !== 201 || !retry.body.nextQuestion) {
        throw new Error(
          `level-complete setup failed: ${JSON.stringify(retry.body)}`,
        );
      }

      state.liveQuestion3Id = retry.body.nextQuestion.questionId;
    },
  });

  await cap({
    route: 'POST /sessions/live/submit-answer',
    case: 'session-not-found-404',
    method: 'POST',
    path: '/sessions/live/submit-answer',
    jar: 'student',
    csrf: true,
    body: {
      sessionId: ids.unknown,
      questionId: ids.unknown,
      answer: 'x',
    },
    expectStatus: 404,
  });

  await cap({
    route: 'POST /sessions/live/submit-answer',
    case: 'question-not-found-404',
    method: 'POST',
    path: '/sessions/live/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.liveSessionId,
      questionId: ids.unknown,
      answer: 'x',
    }),
    expectStatus: 404,
  });

  await cap({
    route: 'POST /sessions/live/submit-answer',
    case: 'validation-400',
    method: 'POST',
    path: '/sessions/live/submit-answer',
    jar: 'student',
    csrf: true,
    body: (s) => ({
      sessionId: s.liveSessionId,
      questionId: s.liveQuestion3Id,
      answer: '',
    }),
    expectStatus: 400,
  });

  await cap({
    route: 'POST /sessions/live/start',
    case: 'not-found-404',
    method: 'POST',
    path: '/sessions/live/start',
    jar: 'student',
    csrf: true,
    body: { topicId: ids.unknown },
    expectStatus: 404,
  });

  await cap({
    route: 'POST /sessions/live/start',
    case: 'validation-400',
    method: 'POST',
    path: '/sessions/live/start',
    jar: 'student',
    csrf: true,
    body: { topicId: 'not-an-id' },
    expectStatus: 400,
  });

  await cap({
    route: 'POST /sessions/live/start',
    case: 'ai-invalid-503',
    method: 'POST',
    path: '/sessions/live/start',
    jar: 'student',
    csrf: true,
    body: { topicId: ids.topic },
    expectStatus: 503,
    note: 'Stubbed provider returns non-JSON for question generation.',
    setup: () => setAiMode('invalid-json'),
  });

  await cap({
    route: 'POST /sessions/live/start',
    case: 'ai-empty-503',
    method: 'POST',
    path: '/sessions/live/start',
    jar: 'student',
    csrf: true,
    body: { topicId: ids.topic },
    expectStatus: 503,
    note: 'Stubbed provider returns an empty completion.',
    setup: () => setAiMode('empty'),
  });

  await cap({
    route: 'POST /sessions/live/start',
    case: 'ai-http-error-500',
    method: 'POST',
    path: '/sessions/live/start',
    jar: 'student',
    csrf: true,
    body: { topicId: ids.topic },
    expectStatus: 500,
    note: 'Stubbed provider HTTP failure surfaces as unhandled 500.',
    setup: () => setAiMode('http-error'),
  });

  await cap({
    route: 'GET /sessions/live/in-progress',
    case: 'success',
    method: 'GET',
    path: '/sessions/live/in-progress',
    jar: 'student',
    expectStatus: 200,
  });

  await cap({
    route: 'GET /sessions/live/in-progress',
    case: 'unauth-401',
    method: 'GET',
    path: '/sessions/live/in-progress',
    jar: 'fresh',
    expectStatus: 401,
  });

  await cap({
    route: 'DELETE /sessions/live/{id}',
    case: 'success',
    method: 'DELETE',
    path: (s) => `/sessions/live/${s.liveSessionId}`,
    jar: 'student',
    csrf: true,
    expectStatus: 204,
  });

  await cap({
    route: 'DELETE /sessions/live/{id}',
    case: 'not-found-404',
    method: 'DELETE',
    path: `/sessions/live/${ids.unknown}`,
    jar: 'student',
    csrf: true,
    expectStatus: 404,
  });

  await cap({
    route: 'DELETE /sessions/live/{id}',
    case: 'invalid-id-400',
    method: 'DELETE',
    path: '/sessions/live/not-an-id',
    jar: 'student',
    csrf: true,
    expectStatus: 400,
  });

  // ------------------------------------------------------------ ai logs
  await cap({
    route: 'GET /ai/logs',
    case: 'success',
    method: 'GET',
    path: '/ai/logs',
    jar: 'admin',
    expectStatus: 200,
    note: 'Six logs: five generations plus the orphan from ai-invalid-503.',
  });

  await cap({
    route: 'GET /ai/logs',
    case: 'success-pagination',
    method: 'GET',
    path: '/ai/logs?page=2&limit=4',
    jar: 'admin',
    expectStatus: 200,
  });

  await cap({
    route: 'GET /ai/logs',
    case: 'validation-400',
    method: 'GET',
    path: '/ai/logs?limit=0',
    jar: 'admin',
    expectStatus: 400,
  });

  await cap({
    route: 'GET /ai/logs',
    case: 'role-403',
    method: 'GET',
    path: '/ai/logs',
    jar: 'student',
    expectStatus: 403,
  });

  await cap({
    route: 'GET /ai/logs',
    case: 'unauth-401',
    method: 'GET',
    path: '/ai/logs',
    jar: 'fresh',
    expectStatus: 401,
  });
}
