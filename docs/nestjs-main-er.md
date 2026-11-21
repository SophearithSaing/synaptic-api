# NestJS main: actual schema ER diagram

Open `nestjs-main-er.drawio` in [draw.io](https://app.diagrams.net/) using
**File → Open From → Device**. The file is uncompressed, editable draw.io XML.

Source: the `synaptic-api-main` worktree, branch `main`, commit
`928e2725d8f990d90671b448b7e30cab62b8ab63`.
No Go rewrite models or proposed schema changes are included.

## Pages

The file was regenerated from scratch with fresh cells and connectors, without
reusing previous edge routing or waypoints. Related relationships are separated
into focused pages to reduce crossings:

1. **Authentication and content**: users, refresh sessions, categories, topics
   and question sets; declared secondary indexes.
2. **Session ownership**: regular and live sessions, students and topics.
3. **Attempts and evaluations**: attempts/evaluations and their references.
4. **Live questions and AI logs**: live-question and AI-log references.
5. **Embedded shapes and logical links**: `Question`, `Answer`,
   `OverallEvaluation` and string-based logical links.
6. **Whole application ER**: all 11 collections with their full field lists,
   all 18 ObjectId references and the logical `attemptIds` string link.

All 11 collections are included with their declared fields, requiredness,
enums and defaults. All 18 declared ObjectId references appear once across the
focused collection pages and again together on the whole-application page.
Compact reference-only tables repeat the same collection across pages, not
new entities.

The whole-application page uses individually routed orthogonal connectors
with separate target attachment points. No two relationships share a line
segment. Crossings use arc-style line jumps rather than junctions. These
routes are saved explicitly; moving tables may require rerouting connectors.

Entities use editable list/table containers with one row per field. `PK` marks
the document key; `REF` marks a stored ObjectId reference, not an enforced
foreign key. Connectors attach directly to the reference and target `_id`
rows. Crow's-foot notation shows zero-to-many children and either one or
zero-to-one parent per child. Labels also state the target multiplicity.
None of the reference fields is unique.

On page 1, `?` means the Mongoose property is not marked `required`, regardless
of TypeScript optionality. Mongoose may still supply defaults, including array
defaults. On page 2, `?` follows the TypeScript interface declaration.

All collection schemas enable timestamps. Each collection includes rows for
the Mongoose-generated `_id`, `createdAt`, `updatedAt` and default `__v`.
`User` uses the default collection name `users`; all other collection names
are explicit.

## Source mapping

Paths are relative to the NestJS `main` worktree.

| Model / shape              | Source                                               |
| -------------------------- | ---------------------------------------------------- |
| User                       | `src/auth/schemas/user.schema.ts`                    |
| AuthSession                | `src/auth/schemas/auth-session.schema.ts`            |
| Category                   | `src/categories/schemas/category.schema.ts`          |
| Topic                      | `src/topics/schemas/topic.schema.ts`                 |
| QuestionSet, Question      | `src/questions/schemas/question-set.schema.ts`       |
| Session, OverallEvaluation | `src/sessions/schemas/session.schema.ts`             |
| LiveSession                | `src/sessions/schemas/live-session.schema.ts`        |
| LiveQuestion               | `src/sessions/schemas/live-question.schema.ts`       |
| SetAttempt, Answer         | `src/sessions/schemas/set-attempt.schemas.ts`        |
| SessionEvaluation          | `src/sessions/schemas/session-evaluation.schemas.ts` |
| AiLog                      | `src/ai/schemas/ai-log.schema.ts`                    |

## Important limitations of the actual schemas

- Embedded values use `Object` / `[Object]`, not nested Mongoose schemas.
  The TypeScript interfaces do not enforce their persisted shape.
- `SetAttempt.session` and `.liveSession` are independently optional. There
  is no schema validation requiring exactly one. The same holds for
  `SessionEvaluation`.
- `SessionEvaluation.attemptIds` is a string array with no declared `ref`.
  Its logical connection to attempts is shown dashed, not as an ObjectId ref.
- `Answer.questionId` is a string, not a declared Mongoose reference.
- `QuestionSet` has no session, live-session or owner reference.
- `LiveQuestion` has no question-set reference.
- `AiLog.operation` only declares `question-generation`.
- Secondary indexes shown are schema declarations, not an inspection of live
  database indexes. Existing MongoDB records may differ from these schemas.

This is documentation only: no implementation, schema, database or branch
changes were made to the NestJS worktree.
