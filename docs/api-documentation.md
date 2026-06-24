# Synaptic API Documentation

This document describes the currently implemented HTTP API in `synaptic-api` for
frontend integration.

## Runtime basics

- Default base URL: `http://localhost:3000` unless `PORT` is set.
- Protected routes require `Authorization: Bearer <access_token>`.
- Validation strips and rejects unknown DTO fields.
- CORS allows `CLIENT_URL`, defaulting to `http://localhost:4200`.

## Authentication

JWT-protected requests attach this user shape server-side:

```ts
{
  email: string;
  username: string;
  role: 'user' | 'admin';
  userId: string;
}
```

Admin routes require `role: 'admin'`.

## Endpoint summary

| Method | Path                         | Auth | Role       | Purpose |
| ------ | ---------------------------- | ---- | ---------- | ------- |
| `GET` | `/` | No | Any | Health/welcome response. |
| `POST` | `/auth/register` | No | Any | Register and return JWT. |
| `POST` | `/auth/login` | No | Any | Login and return JWT. |
| `POST` | `/categories/category/create` | Yes | Admin | Create category. |
| `GET` | `/categories/categories` | Yes | User/Admin | List categories. |
| `GET` | `/categories/:id` | Yes | User/Admin | Get category by ID. |
| `POST` | `/topics/create` | Yes | Admin | Create topic. |
| `GET` | `/topics` | Yes | User/Admin | List topics. |
| `GET` | `/topics/:id` | Yes | User/Admin | Get topic by ID. |
| `POST` | `/questions/create` | Yes | Admin | Create question sets. |
| `PATCH` | `/questions/:id` | Yes | Admin | Update question set. |
| `GET` | `/questions/topic/:slug` | Yes | User/Admin | Get question sets by topic slug. |
| `GET` | `/questions/:id` | Yes | User/Admin | Get question set by ID. |
| `POST` | `/sessions/start` | Yes | User/Admin | Start session and return session ID plus level 0 question set. |
| `GET` | `/sessions/in-progress` | Yes | User/Admin | List active sessions for the current user. |
| `POST` | `/sessions/continue` | Yes | User/Admin | Return current-level question set. |
| `POST` | `/sessions/submit-answer` | Yes | User/Admin | Submit answers and receive feedback. |

## Common errors

```json
{
  "message": "Topic not found",
  "error": "Not Found",
  "statusCode": 404
}
```

- `400 Bad Request`: validation failure or invalid Mongo ID.
- `401 Unauthorized`: missing/invalid JWT.
- `403 Forbidden`: non-admin calling admin route.
- `404 Not Found`: referenced resource does not exist.
- `503 Service Unavailable`: AI evaluation unavailable or invalid AI response.

---

## Auth endpoints

### `POST /auth/register`

Request:

```json
{
  "username": "student123",
  "email": "student@example.com",
  "password": "password123"
}
```

Response `201`:

```json
{
  "access_token": "<jwt>"
}
```

Validation:

- `username`: string, min length 3.
- `email`: valid email.
- `password`: string, min length 8.

Important errors:

- `409 Email already exists`
- `409 Username already exists`

### `POST /auth/login`

Request:

```json
{
  "email": "student@example.com",
  "password": "password123"
}
```

Response `201`:

```json
{
  "access_token": "<jwt>"
}
```

Important errors:

- `401 Unauthorized` for invalid credentials.

---

## Category endpoints

### `POST /categories/category/create`

Admin only.

Request:

```json
{
  "title": "Computer Science Concepts",
  "slug": "cs-concepts",
  "description": "Core theories and fundamental CS principles.",
  "icon": "cs-concepts"
}
```

Response `201`:

```json
{
  "id": "<category-id>",
  "title": "Computer Science Concepts",
  "slug": "cs-concepts",
  "description": "Core theories and fundamental CS principles.",
  "icon": "cs-concepts"
}
```

### `GET /categories/categories`

Response `200`:

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

### `GET /categories/:id`

Response `200`: same shape as one category above.

---

## Topic endpoints

### `POST /topics/create`

Admin only.

Request:

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

Response `201`:

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

### `GET /topics`

Response `200`: array of topic response DTOs.

### `GET /topics/:id`

Response `200`: one topic response DTO.

---

## Question set endpoints

Question sets are the rendered quiz payloads. They belong to a topic and a
level. Multiple sets may exist for the same topic/level.

### Question shape

```ts
{
  id: string;
  type: 'mcq' | 'written';
  prompt: string;
  options: Array<{ id: string; text: string }>;
  correctOptionId?: string;
  targetConcepts: string[];
  feedback: { correct: string; incorrect: string };
  rubrics: { keyPoints: string[]; misconceptions: string[] };
}
```

For MCQ answers, submit the selected option `id` as the answer.

### Question set response shape

```ts
{
  id: string;
  topic: string | Topic;
  setType: string;
  level: number;
  questions: Question[];
  createdAt?: string;
  updatedAt?: string;
}
```

### `POST /questions/create`

Admin only. Body is an array of `CreateQuestionSetDto`. To create a single
question set, send an array with one item.

Request:

```json
[
  {
    "topic": "<topic-id>",
    "setType": "practice",
    "level": 0,
    "questions": [
      {
        "id": "q1",
        "type": "mcq",
        "prompt": "Which memory area stores function call frames?",
        "options": [
          { "id": "a", "text": "Heap" },
          { "id": "b", "text": "Stack" }
        ],
        "correctOptionId": "b",
        "targetConcepts": ["stack-memory"],
        "feedback": {
          "correct": "Correct. The stack stores call frames.",
          "incorrect": "Review stack vs heap memory."
        },
        "rubrics": {
          "keyPoints": ["Stack stores function call frames"],
          "misconceptions": ["Heap stores call frames"]
        }
      }
    ]
  }
]
```

Response `201`: array of question set responses.

### `PATCH /questions/:id`

Admin only. Body supports partial question set fields.

Response `200`: updated question set response.

### `GET /questions/topic/:slug?populateTopic=true`

Returns question sets for a topic slug.

- `populateTopic=true`: topic is populated.
- Any other value or omitted: topic is the topic ID string.

Response `200`: array of question set responses.

### `GET /questions/:id?populateTopic=true`

Returns one question set by ID.

- default behavior populates topic.
- `populateTopic=false` returns topic as an ID string.

Response `200`: question set response.

---

## Session endpoints

### `POST /sessions/start`

Starts a learning session for the authenticated user and returns the created
session ID plus a level `0` question set for the selected topic.

Request:

```json
{
  "topicId": "<topic-id>"
}
```

Response `201`:

```json
{
  "sessionId": "<session-id>",
  "questionSet": {
    "id": "<question-set-id>",
    "topic": "<topic-id>",
    "setType": "practice",
    "level": 0,
    "questions": []
  }
}
```

Side effects:

- Creates a `Session` with `currentLevel: 0` and `status: "active"`.

### `GET /sessions/in-progress`

Returns active sessions for the authenticated user, sorted by most recently
updated first.

Response `200`:

```json
[
  {
    "id": "<session-id>",
    "student": "<user-id>",
    "topic": {
      "_id": "<topic-id>",
      "title": "Memory Management",
      "slug": "memory-management",
      "description": "Understanding stack, heap, and garbage collection.",
      "icon": "memory-management",
      "tags": ["systems", "runtime"],
      "category": "<category-id>"
    },
    "currentLevel": 3,
    "status": "active",
    "overallEvaluation": {
      "summary": "Completed through level 10 with 0.9 average score.",
      "stengths": ["stack-memory"],
      "weakness": [],
      "recommendations": []
    },
    "startAt": "2026-06-21T00:00:00.000Z",
    "createdAt": "2026-06-21T00:00:00.000Z",
    "updatedAt": "2026-06-21T00:00:00.000Z"
  }
]
```

Use the returned `id` as `sessionId` for `/sessions/continue` and
`/sessions/submit-answer`.

### `POST /sessions/continue`

Returns a question set for the session's current level.

Request:

```json
{
  "sessionId": "<session-id>"
}
```

Response `201`: question set response.

Important errors:

- `404 Session not found`
- `404 Question set not found`

### `POST /sessions/submit-answer`

Submits answers for a question set. MCQ answers are evaluated locally. Written
answers are batched and evaluated by AI in one request.

Request:

```json
{
  "sessionId": "<session-id>",
  "questionSetId": "<question-set-id>",
  "answers": [
    {
      "questionId": "q1",
      "answer": "b"
    },
    {
      "questionId": "q2",
      "answer": "The stack stores function call frames and local variables."
    }
  ]
}
```

Validation:

- `sessionId`: required Mongo ObjectId.
- `questionSetId`: required Mongo ObjectId.
- `answers`: required non-empty array.
- `answers[].questionId`: required non-empty string.
- `answers[].answer`: required non-empty string.

Response `201`:

```json
{
  "attempt": {
    "id": "<set-attempt-id>",
    "user": "<user-id>",
    "session": "<session-id>",
    "topic": "<topic-id>",
    "questionSet": "<question-set-id>",
    "level": 0,
    "answers": [
      {
        "id": "ans-q1",
        "questionId": "q1",
        "questionType": "mcq",
        "answer": "b",
        "correctAnswer": "b",
        "score": 1,
        "feedback": "Correct. The stack stores call frames.",
        "targetConcepts": ["stack-memory"],
        "strength": ["stack-memory"],
        "weakness": [],
        "evaluatedBy": "system"
      },
      {
        "id": "ans-q2",
        "questionId": "q2",
        "questionType": "written",
        "answer": "The stack stores function call frames and local variables.",
        "correctAnswer": "Stack stores function call frames; Stack stores local variables",
        "score": 0.9,
        "feedback": "Good explanation of stack usage.",
        "targetConcepts": ["stack-memory"],
        "strength": ["stack-memory"],
        "weakness": [],
        "evaluatedBy": "ai"
      }
    ],
    "setScore": 1,
    "passed": true,
    "strength": ["stack-memory"],
    "weakness": [],
    "submittedAt": "2026-06-21T00:00:00.000Z",
    "evaluatedAt": "2026-06-21T00:00:00.000Z",
    "createdAt": "2026-06-21T00:00:00.000Z",
    "updatedAt": "2026-06-21T00:00:00.000Z"
  },
  "nextQuestionSet": {
    "id": "<next-question-set-id>",
    "topic": "<topic-id>",
    "setType": "practice",
    "level": 1,
    "questions": []
  }
}
```

`nextQuestionSet` is `null` when:

- the student does not pass, or
- no question set exists for `submittedQuestionSet.level + 1`.

If the student passes any submitted set, the API tries to return the next set
for the same session topic. If the submitted set is also the session's current
level, the session `currentLevel` is incremented.

Progress/evaluation rules:

- Passing threshold is `setScore >= 0.8`.
- Scores are rounded to 1 decimal place.
- Passing a set whose level equals `session.currentLevel` increments
  `currentLevel` by `1`.
- When the completed level is divisible by `10`, a `SessionEvaluation` is
  created:
  - level `10` creates range `0-10`.
  - level `20` creates range `11-20`.
  - level `30` creates range `21-30`.
- `session.overallEvaluation` is updated from all session evaluations.

Important errors:

- `400 Question not found in question set`
- `401 Unauthorized`
- `404 Session not found`
- `404 Question set not found`
- `503 AI is not configured`
- `503 AI response was empty`
- `503 AI response was invalid`
- `503 AI response was incomplete`

---

## Data model reference

### Session

```ts
{
  _id: string;
  student: string;
  topic: string;
  currentLevel: number;
  status: string;
  overallEvaluation?: {
    summary: string;
    stengths: string[];
    weakness: string[];
    recommendations: string[];
  };
  startAt?: string;
  finishAt?: string;
  createdAt: string;
  updatedAt: string;
}
```

### SetAttempt

```ts
{
  _id: string;
  user: string;
  session: string;
  topic: string;
  questionSet: string;
  level: number;
  answers: Answer[];
  setScore: number;
  passed: boolean;
  strength: string[];
  weakness: string[];
  aiSummary?: string;
  submittedAt: string;
  evaluatedAt: string;
  createdAt: string;
  updatedAt: string;
}
```

### Answer

```ts
{
  id: string;
  questionId: string;
  questionType: 'mcq' | 'written';
  answer: string;
  correctAnswer: string;
  score: number;
  feedback: string;
  targetConcepts: string[];
  strength: string[];
  weakness: string[];
  evaluatedBy: 'system' | 'ai';
}
```

### SessionEvaluation

```ts
{
  _id: string;
  student: string;
  session: string;
  topic: string;
  fromLevel: number;
  toLevel: number;
  overallScore: number;
  summary: string;
  stength: string[];
  weakness: string[];
  recommendation: string[];
  attemptIds: string[];
  createdAt: string;
  updatedAt: string;
}
```

## Suggested frontend flow

1. Register or login and store `access_token`.
2. Fetch topics with `GET /topics`.
3. Start a session with `POST /sessions/start` and store `sessionId`.
4. Render the returned `questionSet`.
5. Submit answers with `POST /sessions/submit-answer` using `sessionId`.
6. Show `attempt.answers` feedback.
7. If `nextQuestionSet` is not `null`, render it next.
8. If the user comes back later, call `GET /sessions/in-progress` to list
   active sessions, then call `POST /sessions/continue` with the chosen session
   ID to fetch its current-level question set.
