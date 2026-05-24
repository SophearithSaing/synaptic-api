# Codebase Analysis

This project is a NestJS + MongoDB/Mongoose API for an AI-driven learning platform focused on computing education.

## Validation Status

- `npm run build` passes.
- `npm test -- --runInBand` passes.
- Only one unit test currently exists: `src/app.controller.spec.ts`.

## Architecture Overview

### Main Modules

#### `AuthModule`

Handles user registration and login.

- Registers users with email/password.
- Hashes passwords using `bcrypt`.
- Issues JWT access tokens.
- Creates a `StudentModel` when a new user registers.

Key files:

- `src/auth/auth.module.ts`
- `src/auth/auth.controller.ts`
- `src/auth/auth.service.ts`
- `src/auth/strategies/jwt.strategy.ts`
- `src/auth/guards/jwt-auth.guard.ts`
- `src/auth/schemas/user.schema.ts`

#### `AiModule`

Handles interaction with AI providers.

- Supports Gemini and Claude providers.
- Uses Zod to validate AI-generated JSON.
- Provides question generation and answer evaluation.

Key files:

- `src/ai/ai.module.ts`
- `src/ai/ai.service.ts`
- `src/ai/types/ai.types.ts`
- `src/ai/utils/ai.utils.ts`

#### `TopicsModule`

Handles topic and topic-category management.

- Creates categories.
- Creates topics.
- Fetches a topic by ID.

Key files:

- `src/topics/topics.module.ts`
- `src/topics/topics.controller.ts`
- `src/topics/topics.service.ts`
- `src/topics/dto/create-topic.dto.ts`
- `src/topics/schemas/topic.schema.ts`
- `src/topics/schemas/topic-category.schema.ts`
- `src/topics/schemas/topic-progress.schema.ts`

#### `SessionsModule`

Handles learning sessions.

- Starts sessions for a topic.
- Generates AI questions.
- Stores question sets.
- Accepts student answers.
- Evaluates answers using AI.
- Updates topic mastery and overall student level.

Key files:

- `src/sessions/sessions.module.ts`
- `src/sessions/sessions.controller.ts`
- `src/sessions/sessions.service.ts`

#### `SeedModule`

Seeds development data on module initialization.

- Topic categories.
- Topics.
- A default development student model.

Key files:

- `src/seed/seed.module.ts`
- `src/seed/seed.service.ts`

## Database Models

The project uses Mongoose schemas for persistence.

### `User`

Represents an authenticated user.

Fields:

- `email`
- `password`

File:

- `src/auth/schemas/user.schema.ts`

### `StudentModel`

Represents a learner profile.

Fields:

- `userId`
- `overallLevel`
- `topicMastery`

File:

- `src/students/schemas/student-model.schema.ts`

### `TopicCategory`

Represents a grouping for topics.

Fields:

- `title`
- `slug`
- `description`
- `icon`

File:

- `src/topics/schemas/topic-category.schema.ts`

### `Topic`

Represents a learning topic.

Fields:

- `title`
- `slug`
- `description`
- `icon`
- `category`

File:

- `src/topics/schemas/topic.schema.ts`

### `TopicProgress`

Tracks user progress for a topic.

Fields:

- `userId`
- `topic`
- `history`
- `currentLevel`
- `status`

File:

- `src/topics/schemas/topic-progress.schema.ts`

### `Question`

Represents a generated question.

Fields:

- `type`
- `text`
- `options`
- `correctOption`
- `idealAnswerPoints`
- `studentAnswer`
- `isCorrect`
- `score`
- `feedback`
- `difficulty`

File:

- `src/questions/schemas/question.schema.ts`

### `QuestionSet`

Represents a learning session's generated question set.

Fields:

- `userId`
- `topic`
- `questions`
- `score`
- `difficulty`
- `weakConcepts`
- `strongConcepts`
- `feedback`

File:

- `src/questions/schemas/question-set.schema.ts`

## Strengths

- Clean NestJS module separation.
- Mongoose schemas are simple and understandable.
- Passwords are hashed before storage.
- JWT authentication is implemented.
- AI JSON output is validated with Zod.
- Seed data provides useful development defaults.
- The project currently builds successfully.
- Existing test suite passes.

## Issues and Recommendations

### 1. Authenticated endpoints trust `userId` from request body

`SessionsController.startSession()` currently accepts `userId` from the request body.

```ts
@Body('userId') userId: string
```

This is unsafe because any authenticated user can start a session for another user by passing a different `userId`.

Recommended fix:

- Read `userId` from the JWT-authenticated request object instead of the body.

Example direction:

```ts
@Post('start')
async startSession(
  @Req() req: RequestWithUser,
  @Body('topicId') topicId: string,
  @Body('provider') provider?: AiProvider,
) {
  return this.sessionsService.startSession(
    topicId,
    req.user.userId,
    provider,
  );
}
```

### 2. No ownership check when submitting a session

`submitSession(id, answers)` fetches a `QuestionSet` by ID but does not verify that the requesting user owns the session.

Risk:

- A user could submit answers for another user's session if they know the session ID.

Recommended fix:

- Pass the authenticated user ID into `submitSession()`.
- Query by both `_id` and `userId`.

Example direction:

```ts
const questionSet = await this.questionSetModel
  .findOne({ _id: id, userId })
  .populate('questions');
```

### 3. Topics creation is not role-restricted

The topic creation endpoints are protected by JWT, but any authenticated user can create categories or topics.

Affected endpoints:

```ts
@Post('category/create')
@Post('create')
```

If these are intended to be administrative operations, roles or authorization guards are needed.

Recommended fix:

- Add role support to users.
- Add an admin guard for topic/category mutation endpoints.

### 4. Global validation is permissive

`src/main.ts` currently uses:

```ts
app.useGlobalPipes(new ValidationPipe());
```

Recommended stronger configuration:

```ts
app.useGlobalPipes(
  new ValidationPipe({
    whitelist: true,
    forbidNonWhitelisted: true,
    transform: true,
  }),
);
```

Benefits:

- Removes unexpected fields.
- Rejects unknown fields.
- Enables automatic primitive type conversion where appropriate.

### 5. ObjectId errors are not handled consistently

Routes like this pass IDs directly into Mongoose:

```ts
@Get(':id')
async getTopicById(@Param('id') id: string): Promise<TopicDocument> {
  return this.topicsService.getTopicById(id);
}
```

Invalid MongoDB ObjectIds can produce Mongoose cast errors, potentially returning `500` instead of `400`.

Recommended fix:

- Add ObjectId validation with a custom pipe or DTO validation.

### 6. AI provider input is not DTO-validated

The `provider` field is read directly from the request body in session endpoints.

Risk:

- Invalid provider values may cause runtime errors.

Recommended fix:

- Create DTOs for session start and submit requests.
- Validate `provider` with `@IsEnum(AiProvider)`.

### 7. Session difficulty is saved incorrectly

In `SessionsService.startSession()`, difficulty is calculated from the student model:

```ts
const difficulty = studentModel.overallLevel;
```

But the saved question set currently uses a hardcoded value:

```ts
difficulty: 1,
```

Recommended fix:

```ts
difficulty,
```

### 8. AI question mix does not match project requirements

`GEMINI.md` specifies the desired question mix:

- If `P < 40`: provide 3 MCQs.
- If `40 <= P < 70`: provide 1 MCQ and 2 written questions.
- If `P >= 70`: provide 3 written questions with high technical complexity.

Current prompt only says:

```ts
The questions should be a mix of MCQ and Written types.
```

Recommended fix:

- Add explicit question-mix rules to the AI prompt based on the student's difficulty/level.

### 9. Low test coverage

Only the default app controller test exists.

Important areas lacking tests:

- Auth registration.
- Auth login.
- Duplicate email handling.
- JWT guard behavior.
- Topic creation and fetching.
- Session start flow.
- Answer submission flow.
- AI JSON parsing failures.
- Session ownership checks.
- Student mastery updates.

## Suggested Priority Order

1. Use JWT user identity instead of body `userId`.
2. Add session ownership checks.
3. Add DTOs for session start and submit requests.
4. Strengthen the global validation pipe.
5. Fix saved `QuestionSet.difficulty`.
6. Validate Mongo ObjectIds before Mongoose queries.
7. Align AI question generation with the required question mix.
8. Add tests for auth and session flows.
9. Add admin authorization if topic creation should not be user-accessible.

## Summary

The codebase is clean and functional, with good modular separation and a reasonable first implementation of AI-backed learning sessions. The most important improvements are around authorization, input validation, and test coverage. The current implementation builds and passes tests, but several security-sensitive flows should be tightened before production use.
