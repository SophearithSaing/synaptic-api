# Synaptic API Documentation

This document describes the currently implemented HTTP API in `synaptic-api`.
It is intended as a handoff document for another coding agent integrating with
or extending the backend.

## Runtime basics

- Framework: NestJS.
- Default base URL: `http://localhost:3000` unless `PORT` is set.
- Global validation is enabled in `src/main.ts` with:
  - `whitelist: true` unknown DTO fields are stripped.
  - `forbidNonWhitelisted: true` unknown DTO fields also cause `400` errors.
  - `transform: true` DTO transformation is enabled.
- Authentication uses JWT bearer tokens from the `Authorization` header.
- CORS is enabled in `src/main.ts` for `CLIENT_URL`, defaulting to
  `http://localhost:4200` for local Angular development.

## Authentication

Protected endpoints require:

```http
Authorization: Bearer <access_token>
```

JWT payloads are generated with:

```json
{
  "email": "user@example.com",
  "username": "student123",
  "sub": "<mongo-user-id>"
}
```

At request time, the JWT strategy loads the user from MongoDB and attaches:

```ts
{
  email: string;
  username: string;
  role: 'user' | 'admin';
  userId: string;
}
```

Admin-only routes additionally require the stored user role to be `admin`.
New registered users default to role `user`; there is no public role-management
endpoint in the current API.

## Endpoint summary

| Method | Path                          | Auth | Role       | Purpose                                    |
| ------ | ----------------------------- | ---- | ---------- | ------------------------------------------ |
| `GET`  | `/`                           | No   | Any        | Basic health/welcome response.             |
| `POST` | `/auth/register`              | No   | Any        | Create a user.                             |
| `POST` | `/auth/login`                 | No   | Any        | Return a JWT access token.                 |
| `POST` | `/categories/category/create` | Yes  | Admin      | Create a topic category.                   |
| `GET`  | `/categories/categories`      | Yes  | User/Admin | List all topic categories.                 |
| `GET`  | `/categories/:id`             | Yes  | User/Admin | Fetch a category by Mongo ID.              |
| `POST` | `/topics/create`              | Yes  | Admin      | Create a topic.                            |
| `GET`  | `/topics`                     | Yes  | User/Admin | List all topics.                           |
| `GET`  | `/topics/:id`                 | Yes  | User/Admin | Fetch a topic by Mongo ID.                 |
| `POST` | `/sessions/start`             | Yes  | User/Admin | Generate a 3-question session for a topic. |
| `POST` | `/sessions/:id/submit`        | Yes  | User/Admin | Submit answers for evaluation.             |

## Error behavior

Common NestJS error shapes look like:

```json
{
  "message": "Topic not found",
  "error": "Not Found",
  "statusCode": 404
}
```

Validation failures return `400 Bad Request`. Invalid or missing JWTs return
`401 Unauthorized`. Authenticated non-admin users calling admin routes receive
`403 Forbidden`.

Mongo ID route parameters use `MongoIdPipe`; invalid IDs return `400`.

## Endpoints

### `GET /`

Basic root endpoint implemented by `AppController`.

#### Response

```text
Hello World!
```

The exact string comes from `AppService.getHello()`.

---

### `POST /auth/register`

Registers a new user.

#### Request body

```json
{
  "username": "student123",
  "email": "student@example.com",
  "password": "password123"
}
```

Validation:

- `username`: required string, minimum length 3.
- `email`: valid email.
- `password`: required, minimum length 8.

#### Response `201 Created`

```json
{
  "access_token": "<jwt>"
}
```

#### Important errors

- `409 Conflict` with message `Email already exists` when the email is already
  registered.
- `409 Conflict` with message `Username already exists` when the username is
  already registered.
- `400 Bad Request` for invalid username, email, or password.

#### Usage notes

Registration returns an access token, so clients can immediately authenticate
subsequent protected API requests without calling `/auth/login`.

---

### `POST /auth/login`

Authenticates a user and returns a JWT access token.

#### Request body

```json
{
  "email": "student@example.com",
  "password": "password123"
}
```

Validation:

- `email`: valid email.
- `password`: required, minimum length 8.

#### Response `201 Created`

```json
{
  "access_token": "<jwt>"
}
```

#### Important errors

- `401 Unauthorized` for unknown email or invalid password.

#### Usage notes

Store `access_token` client-side and send it as a bearer token to every
protected endpoint.

---

### `POST /categories/category/create`

Creates a topic category.

#### Auth

- Requires JWT.
- Requires user role `admin`.

#### Request body

```json
{
  "title": "Computer Science Concepts",
  "slug": "cs-concepts",
  "description": "Core theories and fundamental CS principles.",
  "icon": "cs-concepts"
}
```

Validation:

- All fields are required non-empty strings.

#### Response `201 Created`

Returns the category response DTO.

```json
{
  "id": "<category-id>",
  "title": "Computer Science Concepts",
  "slug": "cs-concepts",
  "description": "Core theories and fundamental CS principles.",
  "icon": "cs-concepts"
}
```

#### Important errors

- `401 Unauthorized` if unauthenticated.
- `403 Forbidden` if authenticated but not admin.
- Mongo duplicate errors can occur for duplicate `slug` values; these are not
  currently converted to a friendly API error.

---

### `POST /topics/create`

Creates a topic inside a category.

#### Auth

- Requires JWT.
- Requires user role `admin`.

#### Request body

```json
{
  "title": "Memory Management",
  "slug": "memory-management",
  "description": "Understanding stack, heap, and garbage collection.",
  "icon": "memory-management",
  "tags": ["systems", "runtime"],
  "category": "<category-id>"
}
```

Validation:

- `title`, `slug`, `description`, `icon`: required non-empty strings.
- `tags`: required array of 1 or 2 non-empty strings.
- `category`: required Mongo ObjectId string.

#### Response `201 Created`

Returns the topic response DTO with its category populated.

```json
{
  "id": "<topic-id>",
  "title": "Memory Management",
  "slug": "memory-management",
  "description": "Understanding stack, heap, and garbage collection.",
  "icon": "memory-management",
  "tags": ["systems", "runtime"],
  "category": {
    "id": "<category-id>",
    "title": "Computer Science Concepts",
    "slug": "cs-concepts",
    "description": "Core theories and fundamental CS principles.",
    "icon": "cs-concepts"
  }
}
```

#### Important errors

- `401 Unauthorized` if unauthenticated.
- `403 Forbidden` if authenticated but not admin.
- Mongo duplicate errors can occur for duplicate `slug` values; these are not
  currently converted to a friendly API error.

---

### `GET /topics`

Lists all topics sorted by title.

#### Auth

- Requires JWT.
- Any authenticated role can call it.

#### Response `200 OK`

Each topic is returned as a response DTO with `category` populated.

```json
[
  {
    "id": "<topic-id>",
    "title": "Memory Management",
    "slug": "memory-management",
    "description": "Understanding stack, heap, and garbage collection.",
    "icon": "memory-management",
    "tags": ["systems", "runtime"],
    "category": {
      "id": "<category-id>",
      "title": "Computer Science Concepts",
      "slug": "cs-concepts",
      "description": "Core theories and fundamental CS principles.",
      "icon": "cs-concepts"
    }
  }
]
```

#### Important errors

- `401 Unauthorized` if unauthenticated.

---

### `GET /categories/categories`

Lists all topic categories sorted by title.

#### Auth

- Requires JWT.
- Any authenticated role can call it.

#### Response `200 OK`

```json
[
  {
    "id": "<category-id>",
    "title": "Computer Science Concepts",
    "slug": "cs-concepts",
    "description": "Core theories and fundamental CS principles.",
    "icon": "cs-concepts"
  }
]
```

#### Important errors

- `401 Unauthorized` if unauthenticated.

---

### `GET /categories/:id`

Fetches a category by Mongo ObjectId.

#### Auth

- Requires JWT.
- Any authenticated role can call it.

#### Path params

- `id`: category Mongo ObjectId.

#### Response `200 OK`

```json
{
  "id": "<category-id>",
  "title": "Computer Science Concepts",
  "slug": "cs-concepts",
  "description": "Core theories and fundamental CS principles.",
  "icon": "cs-concepts"
}
```

#### Important errors

- `400 Bad Request` for invalid Mongo ID format.
- `401 Unauthorized` if unauthenticated.
- `404 Not Found` with message `Category not found` if the ID is valid but no
  category exists.

---

### `GET /topics/:id`

Fetches a topic by Mongo ObjectId.

#### Auth

- Requires JWT.
- Any authenticated role can call it.

#### Path params

- `id`: topic Mongo ObjectId.

#### Response `200 OK`

The topic is returned as a response DTO with `category` populated.

```json
{
  "id": "<topic-id>",
  "title": "Memory Management",
  "slug": "memory-management",
  "description": "Understanding stack, heap, and garbage collection.",
  "icon": "memory-management",
  "tags": ["systems", "runtime"],
  "category": {
    "id": "<category-id>",
    "title": "Computer Science Concepts",
    "slug": "cs-concepts",
    "description": "Core theories and fundamental CS principles.",
    "icon": "cs-concepts"
  }
}
```

#### Important errors

- `400 Bad Request` for invalid Mongo ID format.
- `401 Unauthorized` if unauthenticated.
- `404 Not Found` with message `Topic not found` if the ID is valid but no
  topic exists.

---

### `POST /sessions/start`

Starts a learning session for the authenticated user on a topic. The server
currently uses difficulty `0` and asks the AI provider to generate exactly three
questions.

#### Auth

- Requires JWT.
- Any authenticated role can call it.

#### Request body

```json
{
  "topicId": "<topic-id>",
  "provider": "gemini"
}
```

Validation:

- `topicId`: required Mongo ObjectId.
- `provider`: optional enum, either `gemini` or `claude`.

If `provider` is omitted, the AI service defaults to `gemini`.

#### Response `201 Created`

Returns the created `QuestionSet` with `topic` and `questions` populated.

```json
{
  "_id": "<question-set-id>",
  "userId": "<user-id>",
  "topic": {
    "_id": "<topic-id>",
    "title": "Memory Management",
    "slug": "memory-management",
    "description": "Understanding stack, heap, and garbage collection.",
    "icon": "memory-management",
    "tags": ["systems", "runtime"],
    "category": "<category-id>",
    "createdAt": "2026-06-03T00:00:00.000Z",
    "updatedAt": "2026-06-03T00:00:00.000Z",
    "__v": 0
  },
  "questions": [
    {
      "_id": "<question-id>",
      "type": "mcq",
      "text": "Which area of memory typically stores function call frames?",
      "options": ["Heap", "Stack", "Code segment", "Disk"],
      "correctOption": "Stack",
      "idealAnswerPoints": ["Stack stores call frames"],
      "difficulty": 1,
      "score": 0,
      "createdAt": "2026-06-03T00:00:00.000Z",
      "updatedAt": "2026-06-03T00:00:00.000Z",
      "__v": 0
    }
  ],
  "score": 0,
  "difficulty": 1,
  "weakConcepts": [],
  "strongConcepts": [],
  "createdAt": "2026-06-03T00:00:00.000Z",
  "updatedAt": "2026-06-03T00:00:00.000Z",
  "__v": 0
}
```

#### Question mix by difficulty

The AI prompt chooses question type mix based on session difficulty:

- `< 40`: exactly 3 MCQs.
- `40` to `69`: exactly 1 MCQ and 2 written questions.
- `>= 70`: exactly 3 written questions with high technical complexity.

#### Side effects

- Creates a `Question` document for each generated question.
- Creates a `QuestionSet` document for the session.
- Does not currently update persisted student progress.

#### Important errors

- `400 Bad Request` for invalid body or provider.
- `401 Unauthorized` if unauthenticated.
- `404 Not Found` with message `Topic not found` if the topic does not exist.
- `500 Internal Server Error` if the AI provider call fails or returns invalid
  JSON/schema data.

---

### `POST /sessions/:id/submit`

Submits answers for a previously generated question set. The server evaluates
answers with the selected AI provider, stores per-question feedback, and updates
the question set summary.

#### Auth

- Requires JWT.
- Any authenticated role can call it.
- The session must belong to the authenticated user.

#### Path params

- `id`: question set Mongo ObjectId returned by `/sessions/start`.

#### Request body

```json
{
  "answers": [
    {
      "questionId": "<question-id>",
      "studentAnswer": "Stack"
    },
    {
      "questionId": "<question-id>",
      "studentAnswer": "The heap stores dynamically allocated objects."
    }
  ],
  "provider": "gemini"
}
```

Validation:

- `answers`: required non-empty array.
- Each answer requires:
  - `questionId`: Mongo ObjectId.
  - `studentAnswer`: non-empty string.
- `provider`: optional enum, either `gemini` or `claude`.

If `provider` is omitted, the AI service defaults to `gemini`.

#### Response `201 Created`

```json
{
  "evaluation": {
    "totalScore": 85,
    "critique": "Strong understanding with minor terminology gaps.",
    "weakConcepts": ["Garbage collection details"],
    "strongConcepts": ["Stack vs heap", "Dynamic allocation"],
    "questionEvaluations": [
      {
        "questionId": "<question-id>",
        "score": 100,
        "isCorrect": true,
        "feedback": "Correct. The stack stores function call frames."
      }
    ]
  }
}
```

#### Important errors

- `400 Bad Request` for invalid question set ID, body, question IDs, or
  provider.
- `401 Unauthorized` if unauthenticated.
- `404 Not Found` with message `Question set not found` if the session does not
  exist or belongs to another user.
- `404 Not Found` with message `Question <id> not found in this set` if an
  answer references a question outside the session.
- `500 Internal Server Error` if the AI provider call fails or returns invalid
  JSON/schema data.

## Data model reference

### User

```ts
{
  _id: string;
  username: string;
  email: string;
  password: string; // hashed, never returned by auth endpoints
  role: 'user' | 'admin';
  createdAt: string;
  updatedAt: string;
}
```

### Category

```ts
{
  _id: string;
  title: string;
  slug: string;
  description: string;
  icon: string;
  createdAt: string;
  updatedAt: string;
}
```

### Topic

```ts
{
  _id: string;
  title: string;
  slug: string;
  description: string;
  icon: string;
  tags: string[];
  category: string | Category;
  createdAt: string;
  updatedAt: string;
}
```

### Question

```ts
{
  _id: string;
  type: 'mcq' | 'written';
  text: string;
  options: string[];
  correctOption?: string;
  idealAnswerPoints: string[];
  studentAnswer?: string;
  isCorrect?: boolean;
  score: number;
  feedback?: string;
  difficulty: number;
  createdAt: string;
  updatedAt: string;
}
```

### QuestionSet

```ts
{
  _id: string;
  userId: string;
  topic: string | Topic;
  questions: Array<string | Question>;
  score: number;
  difficulty: number;
  weakConcepts: string[];
  strongConcepts: string[];
  feedback?: string;
  createdAt: string;
  updatedAt: string;
}
```

## Suggested client flow

1. Register a user with `POST /auth/register` and store the returned
   `access_token`.
2. Alternatively, login with `POST /auth/login` and store `access_token` for an
   existing account.
3. Call `GET /topics` to obtain the available topics and select a topic ID.
4. Start a session with `POST /sessions/start`.
5. Render the returned `questions`.
   - For `mcq`, render `options` and submit the chosen option text/string.
   - For `written`, render a text input/textarea.
6. Submit all answers with `POST /sessions/:id/submit`.
7. Use the returned `evaluation` for feedback.

## Seeded content

`SeedService` automatically creates categories and topics on module init if none
exist.

Seeded categories:

- `cs-concepts`: Computer Science Concepts.
- `tech-stacks`: Languages & Tech Stacks.
- `ops-infra`: Operations & Infrastructure.

Seeded topics:

- Memory Management: `systems`, `runtime`
- Concurrency: `systems`, `parallelism`
- Computer Networking: `networking`, `protocols`
- Distributed Systems: `distributed`, `networking`
- Graph Theory: `algorithms`, `graphs`
- Node.js: `backend`, `runtime`
- Go: `systems`, `backend`
- Rust Fundamentals: `systems`, `memory-safety`
- Go Concurrency: `concurrency`, `backend`
- Hyperledger Fabric: `blockchain`, `enterprise`
- Containerization (Docker): `containers`, `devops`
- CI/CD Pipelines: `automation`, `devops`
- Kubernetes: `orchestration`, `containers`

## Integration gaps for another agent to consider

- Add a student profile/progress endpoint for current user.
- Add admin user role management or a documented seed admin path.
- Consider tightening CORS origins per deployment environment.
- Consider hiding `correctOption` and `idealAnswerPoints` from
  `/sessions/start` responses to avoid exposing answers before submission.
- Convert duplicate slug Mongo errors into friendly `409 Conflict` responses.
- Consider returning `200 OK` for login/register/submit if preferred; Nest's
  default for `POST` currently returns `201 Created` unless changed with
  `@HttpCode`.
