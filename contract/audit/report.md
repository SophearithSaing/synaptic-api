# BSON audit report

- Generated: 2026-09-27T12:56:05.546Z (run timestamp)
- Target: database `synaptic` (read-only audit)
- Env file: `.env` (path elided; credentials never read into output)
- Method: countDocuments, aggregate, find (projections),
  distinct, listIndexes only; no writes of any kind.
- Contents: aggregate counts, field names, BSON types, and
  index specs only — no user data, tokens, or content values.

Collections present (12):
`aiLogs`, `authSessions`, `authsessions`, `categories`, `liveQuestions`, `liveSessions`, `questionSets`, `sessionEvaluations`, `sessions`, `setAttempts`, `topics`, `users`

## users

### Counts

- documents: **3**
- documents missing __v: **0**

### Indexes

| name | key | unique | expireAfterSeconds |
| --- | --- | --- | --- |
| _id_ | `{"_id":1}` | false |  |
| username_1 | `{"username":1}` | true |  |
| email_1 | `{"email":1}` | true |  |

### Duplicate natural keys

| natural key | duplicate groups | documents in groups |
| --- | ---: | ---: |
| email (exact) | 0 | 0 |
| username (case-insensitive) | 0 | 0 |

### Dangling references

_none_

### Enum distributions

- role: `admin` × 1, `user` × 2

### Malformed embedded questions

_none found_

### Field shape (name + BSON type frequency)

| field | BSON type | documents |
| --- | --- | ---: |
| __v | int | 3 |
| _id | objectId | 3 |
| createdAt | date | 3 |
| email | string | 3 |
| password | string | 3 |
| role | string | 3 |
| updatedAt | date | 3 |
| username | string | 3 |

## authSessions

### Counts

- documents: **1**
- documents missing __v: **0**
- revoked: **0**
- expired (by expiresAt): **0**

### Indexes

| name | key | unique | expireAfterSeconds |
| --- | --- | --- | --- |
| _id_ | `{"_id":1}` | false |  |
| userId_1 | `{"userId":1}` | false |  |
| expiresAt_1 | `{"expiresAt":1}` | false | 0 |

### Duplicate natural keys

| natural key | duplicate groups | documents in groups |
| --- | ---: | ---: |
| sessions per user (>1 total) | 0 | 0 |
| active sessions per user (>1) | 0 | 0 |

### Dangling references

- userId -> users: **0**

### Enum distributions

_none_

### Malformed embedded questions

_none found_

### Field shape (name + BSON type frequency)

| field | BSON type | documents |
| --- | --- | ---: |
| __v | int | 1 |
| _id | objectId | 1 |
| createdAt | date | 1 |
| expiresAt | date | 1 |
| refreshTokenHash | string | 1 |
| updatedAt | date | 1 |
| userId | objectId | 1 |

## categories

### Counts

- documents: **5**
- documents missing __v: **0**

### Indexes

| name | key | unique | expireAfterSeconds |
| --- | --- | --- | --- |
| _id_ | `{"_id":1}` | false |  |
| slug_1 | `{"slug":1}` | true |  |

### Duplicate natural keys

| natural key | duplicate groups | documents in groups |
| --- | ---: | ---: |
| slug | 0 | 0 |

### Dangling references

_none_

### Enum distributions

_none_

### Malformed embedded questions

_none found_

### Field shape (name + BSON type frequency)

| field | BSON type | documents |
| --- | --- | ---: |
| __v | int | 5 |
| _id | objectId | 5 |
| createdAt | date | 5 |
| description | string | 5 |
| icon | string | 5 |
| slug | string | 5 |
| title | string | 5 |
| updatedAt | date | 5 |

## topics

### Counts

- documents: **23**
- documents missing __v: **0**
- tags outside 1..2: **0**
- category refs stored as string: **10**
- string category refs resolving to a category: **4**
- string category refs truly dangling: **6**

### Indexes

| name | key | unique | expireAfterSeconds |
| --- | --- | --- | --- |
| _id_ | `{"_id":1}` | false |  |
| slug_1 | `{"slug":1}` | true |  |

### Duplicate natural keys

| natural key | duplicate groups | documents in groups |
| --- | ---: | ---: |
| slug | 0 | 0 |

### Dangling references

- category -> categories (raw lookup): **10**

### Enum distributions

_none_

### Malformed embedded questions

_none found_

### Field shape (name + BSON type frequency)

| field | BSON type | documents |
| --- | --- | ---: |
| __v | int | 23 |
| _id | objectId | 23 |
| category | objectId | 13 |
| category | string | 10 |
| createdAt | date | 23 |
| description | string | 23 |
| icon | string | 23 |
| slug | string | 23 |
| tags | array | 23 |
| title | string | 23 |
| updatedAt | date | 23 |

## questionSets

### Counts

- documents: **1377**
- documents missing __v: **0**
- embedded questions scanned: **4131**
- question sets with malformed questions: **1172**

### Indexes

| name | key | unique | expireAfterSeconds |
| --- | --- | --- | --- |
| _id_ | `{"_id":1}` | false |  |

### Duplicate natural keys

| natural key | duplicate groups | documents in groups |
| --- | ---: | ---: |
| topic + level + setType | 16 | 37 |

### Dangling references

- topic -> topics: **0**

### Enum distributions

- setType: `live` × 64, `primary` × 1313
- questions.type: `mcq` × 981, `written` × 3150

### Malformed embedded questions

- duplicate-question-id-in-set: **2**
- written-has-correct-option-id: **3120**
- written-has-options: **3120**

### Field shape (name + BSON type frequency)

| field | BSON type | documents |
| --- | --- | ---: |
| __v | int | 1377 |
| _id | objectId | 1377 |
| createdAt | date | 1377 |
| level | int | 1377 |
| questions.correctOptionId | null | 3120 |
| questions.correctOptionId | string | 981 |
| questions.feedback.correct | string | 4131 |
| questions.feedback.evaluationGuidance | string | 1200 |
| questions.feedback.incorrect | string | 4131 |
| questions.feedback.sampleAnswer | string | 1680 |
| questions.feedback | object | 4131 |
| questions.hints | array | 3939 |
| questions.id | string | 4131 |
| questions.options.id | string | 3195 |
| questions.options.text | string | 3195 |
| questions.options | array | 1941 |
| questions.options | null | 2160 |
| questions.prompt | string | 4131 |
| questions.rubrics.keyPoints | array | 4131 |
| questions.rubrics.misconceptions | array | 4131 |
| questions.rubrics | object | 4131 |
| questions.targetConcepts | array | 4131 |
| questions.type | string | 4131 |
| questions | array | 1377 |
| setType | string | 1377 |
| topic | objectId | 1377 |
| updatedAt | date | 1377 |

## sessions

### Counts

- documents: **0**
- documents missing __v: **0**
- with overallEvaluation: **0**

### Indexes

| name | key | unique | expireAfterSeconds |
| --- | --- | --- | --- |
| _id_ | `{"_id":1}` | false |  |

### Duplicate natural keys

| natural key | duplicate groups | documents in groups |
| --- | ---: | ---: |
| student + topic (total) | 0 | 0 |
| student + topic (active) | 0 | 0 |

### Dangling references

- student -> users: **0**
- topic -> topics: **0**

### Enum distributions

- status: 

### Malformed embedded questions

_none found_

### Field shape (name + BSON type frequency)

_none_

## liveSessions

### Counts

- documents: **3**
- documents missing __v: **0**
- with overallEvaluation: **1**

### Indexes

| name | key | unique | expireAfterSeconds |
| --- | --- | --- | --- |
| _id_ | `{"_id":1}` | false |  |

### Duplicate natural keys

| natural key | duplicate groups | documents in groups |
| --- | ---: | ---: |
| student + topic (total) | 0 | 0 |
| student + topic (active) | 0 | 0 |

### Dangling references

- student -> users: **0**
- topic -> topics: **0**

### Enum distributions

- status: `active` × 3

### Malformed embedded questions

_none found_

### Field shape (name + BSON type frequency)

| field | BSON type | documents |
| --- | --- | ---: |
| __v | int | 3 |
| _id | objectId | 3 |
| createdAt | date | 3 |
| currentLevel | int | 3 |
| overallEvaluation.recommendations | array | 1 |
| overallEvaluation.strengths | array | 1 |
| overallEvaluation.summary | string | 1 |
| overallEvaluation.weaknesses | array | 1 |
| overallEvaluation | object | 1 |
| startedAt | date | 3 |
| status | string | 3 |
| student | objectId | 3 |
| topic | objectId | 3 |
| updatedAt | date | 3 |

## liveQuestions

### Counts

- documents: **203**
- documents missing __v: **0**
- with answer: **169**
- embedded questions scanned: **203**
- live questions malformed: **0**

### Indexes

| name | key | unique | expireAfterSeconds |
| --- | --- | --- | --- |
| _id_ | `{"_id":1}` | false |  |

### Duplicate natural keys

| natural key | duplicate groups | documents in groups |
| --- | ---: | ---: |
| session + level + questionNumber (all) | 22 | 50 |
| session + level + questionNumber (pending) | 0 | 0 |
| session + level + questionNumber (not rejected) | 3 | 6 |

### Dangling references

- liveSession -> liveSessions: **28**

### Enum distributions

- status: `accepted` × 1, `passed` × 167, `pending` × 10, `rejected` × 25
- question.type: `mcq` × 159, `written` × 44
- answer.evaluatedBy: `(absent)` × 34, `ai` × 29, `system` × 140

### Malformed embedded questions

_none found_

### Field shape (name + BSON type frequency)

| field | BSON type | documents |
| --- | --- | ---: |
| __v | int | 203 |
| _id | objectId | 203 |
| answer.answer | string | 169 |
| answer.answerText | string | 117 |
| answer.correctAnswer | string | 169 |
| answer.correctAnswerText | string | 117 |
| answer.evaluatedBy | string | 169 |
| answer.feedback | string | 169 |
| answer.id | string | 169 |
| answer.questionId | string | 169 |
| answer.questionPrompt | string | 150 |
| answer.questionType | string | 169 |
| answer.score | double | 14 |
| answer.score | int | 155 |
| answer.strengths | array | 169 |
| answer.targetConcepts | array | 169 |
| answer.weaknesses | array | 169 |
| answeredAt | date | 169 |
| answer | object | 169 |
| createdAt | date | 203 |
| level | int | 203 |
| liveSession | objectId | 203 |
| question.correctOptionId | string | 159 |
| question.feedback.correct | string | 203 |
| question.feedback.incorrect | string | 203 |
| question.feedback | object | 203 |
| question.id | string | 203 |
| question.options.id | string | 477 |
| question.options.text | string | 477 |
| question.options | array | 159 |
| question.prompt | string | 203 |
| question.rubrics.keyPoints | array | 203 |
| question.rubrics.misconceptions | array | 203 |
| question.rubrics | object | 203 |
| question.targetConcepts | array | 203 |
| question.type | string | 203 |
| questionNumber | int | 203 |
| question | object | 203 |
| status | string | 203 |
| updatedAt | date | 203 |

## setAttempts

### Counts

- documents: **145**
- documents missing __v: **0**
- session and liveSession both set: **0**
- neither session nor liveSession set: **0**
- passed: **123**
- failed: **22**

### Indexes

| name | key | unique | expireAfterSeconds |
| --- | --- | --- | --- |
| _id_ | `{"_id":1}` | false |  |

### Duplicate natural keys

| natural key | duplicate groups | documents in groups |
| --- | ---: | ---: |
| session + questionSet | 9 | 21 |
| liveSession + questionSet | 0 | 0 |
| session + level | 9 | 21 |
| liveSession + level | 9 | 18 |

### Dangling references

- user -> users: **0**
- session -> sessions (when set): **81**
- liveSession -> liveSessions (when set): **10**
- topic -> topics: **0**
- questionSet -> questionSets: **0**

### Enum distributions

- answers.evaluatedBy: `ai` × 49, `system` × 386

### Malformed embedded questions

_none found_

### Field shape (name + BSON type frequency)

| field | BSON type | documents |
| --- | --- | ---: |
| __v | int | 145 |
| _id | objectId | 145 |
| answers.answer | string | 435 |
| answers.answerText | string | 135 |
| answers.correctAnswer | string | 435 |
| answers.correctAnswerText | string | 135 |
| answers.evaluatedBy | string | 435 |
| answers.feedback | string | 435 |
| answers.id | string | 435 |
| answers.questionId | string | 435 |
| answers.questionPrompt | string | 168 |
| answers.questionType | string | 435 |
| answers.score | double | 30 |
| answers.score | int | 405 |
| answers.strength | array | 72 |
| answers.strengths | array | 363 |
| answers.targetConcepts | array | 435 |
| answers.weakness | array | 72 |
| answers.weaknesses | array | 363 |
| answers | array | 145 |
| createdAt | date | 145 |
| evaluatedAt | date | 145 |
| level | int | 145 |
| liveSession | objectId | 64 |
| passed | bool | 145 |
| questionSet | objectId | 145 |
| session | objectId | 81 |
| setScore | double | 38 |
| setScore | int | 107 |
| strength | array | 24 |
| strengths | array | 121 |
| submittedAt | date | 145 |
| topic | objectId | 145 |
| updatedAt | date | 145 |
| user | objectId | 145 |
| weakness | array | 24 |
| weaknesses | array | 121 |

## sessionEvaluations

### Counts

- documents: **7**
- documents missing __v: **0**
- session and liveSession both set: **0**
- neither session nor liveSession set: **0**
- distinct attemptIds referenced: **80**
- attemptIds with invalid ObjectId format: **0**
- dangling attemptIds (distinct): **0**

### Indexes

| name | key | unique | expireAfterSeconds |
| --- | --- | --- | --- |
| _id_ | `{"_id":1}` | false |  |

### Duplicate natural keys

| natural key | duplicate groups | documents in groups |
| --- | ---: | ---: |
| session + fromLevel + toLevel | 0 | 0 |
| liveSession + fromLevel + toLevel | 0 | 0 |

### Dangling references

- student -> users: **0**
- session -> sessions (when set): **5**
- liveSession -> liveSessions (when set): **0**
- topic -> topics: **0**

### Enum distributions

_none_

### Malformed embedded questions

_none found_

### Field shape (name + BSON type frequency)

| field | BSON type | documents |
| --- | --- | ---: |
| __v | int | 7 |
| _id | objectId | 7 |
| attemptIds[] | string | 80 |
| attemptIds | array | 7 |
| createdAt | date | 7 |
| fromLevel | int | 7 |
| liveSession | objectId | 2 |
| overallScore | double | 1 |
| overallScore | int | 6 |
| recommendation | array | 1 |
| recommendations[] | string | 3 |
| recommendations | array | 6 |
| session | objectId | 5 |
| stength | array | 1 |
| strengths[] | string | 211 |
| strengths | array | 6 |
| student | objectId | 7 |
| summary | string | 7 |
| toLevel | int | 7 |
| topic | objectId | 7 |
| updatedAt | date | 7 |
| weakness | array | 1 |
| weaknesses[] | string | 3 |
| weaknesses | array | 6 |

## aiLogs

### Counts

- documents: **148**
- documents missing __v: **0**
- unlinked (no liveQuestion): **0**

### Indexes

| name | key | unique | expireAfterSeconds |
| --- | --- | --- | --- |
| _id_ | `{"_id":1}` | false |  |
| createdAt_-1 | `{"createdAt":-1}` | false |  |

### Duplicate natural keys

_none_

### Dangling references

- liveQuestion -> liveQuestions (when set): **0**

### Enum distributions

- operation: `question-generation` × 148

### Malformed embedded questions

_none found_

### Field shape (name + BSON type frequency)

| field | BSON type | documents |
| --- | --- | ---: |
| __v | int | 148 |
| _id | objectId | 148 |
| aiModel | string | 148 |
| createdAt | date | 148 |
| liveQuestion | objectId | 148 |
| operation | string | 148 |
| output | string | 148 |
| prompt | string | 148 |
| updatedAt | date | 148 |

## Findings for the Go BSON mappers

Derived from the measured values above (snapshot-specific).

- Collections present but NOT part of the current 11-collection schema (not audited): `authsessions`.
- `questionSets.setType` contains values outside the current enum (`regular`/`live`): `primary`.
- `liveQuestions.status` contains values outside the current enum: `accepted`.
- `topics.category` is stored as a **string** (not ObjectId) in 10 documents — see the string-ref breakdown in the topics section.
- Embedded questions carry fields absent from the current DTO: `questions[].hints`, `questions[].feedback.sampleAnswer`, `questions[].feedback.evaluationGuidance`.
- Some written questions carry `options`/`correctOptionId` with **null** values (not absent fields) — treat both as omitempty.
- Singular/typo field variants exist alongside the plural forms: `setAttempts.answers.strength`, `setAttempts.answers.weakness`, `setAttempts.strength`, `setAttempts.weakness`, `sessionEvaluations.stength`, `sessionEvaluations.recommendation`, `sessionEvaluations.weakness`.
- Score fields mix **int** and **double** BSON types: `liveQuestions.answer.score`, `setAttempts.answers.score`, `setAttempts.setScore`, `sessionEvaluations.overallScore`.
- Duplicate natural keys exist: `questionSets` topic + level + setType (16 groups / 37 docs); `liveQuestions` session + level + questionNumber (all) (22 groups / 50 docs); `liveQuestions` session + level + questionNumber (not rejected) (3 groups / 6 docs); `setAttempts` session + questionSet (9 groups / 21 docs); `setAttempts` session + level (9 groups / 21 docs); `setAttempts` liveSession + level (9 groups / 18 docs).
- Dangling references exist: `topics` category -> categories (raw lookup) (10); `liveQuestions` liveSession -> liveSessions (28); `setAttempts` session -> sessions (when set) (81); `setAttempts` liveSession -> liveSessions (when set) (10); `sessionEvaluations` session -> sessions (when set) (5).

Structural facts independent of this snapshot:

- `sessionEvaluations.attemptIds` is stored as an array of
  **strings**, not ObjectIds (see schema and shape table).
- Optional refs (`session`/`liveSession` on setAttempts and
  sessionEvaluations, `liveQuestion` on aiLogs) may be absent
  or null; mappers must treat them as omitempty pointers.
- Embedded question shapes are polymorphic on `type`; malformed
  counts above describe exactly which legacy deviations exist.

