#!/usr/bin/env node
/**
 * Read-only BSON audit of the Synaptic database.
 *
 * Connects to the database configured by the Go tree `.env` (or an env
 * file passed as argv[2] / AUDIT_ENV_FILE), runs STRICTLY read-only
 * operations (countDocuments, aggregate, find with projections,
 * distinct, listIndexes) across all 11 collections, and writes an
 * aggregate-counts-only report to contract/audit/report.md.
 *
 * Usage (from the repository root):
 *
 *   node contract/audit/run.mjs [path-to-env-file]
 *
 * Safety:
 * - never prints or persists credentials (the URI is never logged)
 * - collection handles are wrapped in a proxy that throws on any write
 *   method, making the read-only contract executable rather than
 *   conventional
 * - the report contains aggregate counts, field names, BSON types, and
 *   index specs only — no emails, usernames, slugs, prompt/output
 *   content, tokens, or student answers
 */
import { writeFileSync, readFileSync, existsSync } from 'node:fs';
import { join, dirname, resolve, basename } from 'node:path';
import { fileURLToPath } from 'node:url';
import { MongoClient, ObjectId } from 'mongodb';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const worktreeRoot = join(scriptDir, '..', '..');

// ---------------------------------------------------------------- env
const DEFAULT_ENV_FILE = resolve(
  worktreeRoot,
  '..',
  'synaptic-api',
  '.env',
);

/** Minimal .env parser (KEY=value lines, optional quotes, # comments). */
function loadEnvFile(path) {
  const values = {};

  for (const line of readFileSync(path, 'utf8').split('\n')) {
    const match = line.match(/^\s*([A-Z0-9_]+)\s*=\s*(.*)\s*$/);

    if (!match || line.trimStart().startsWith('#')) {
      continue;
    }

    values[match[1]] = match[2].replace(/^['"]|['"]$/g, '');
  }

  return values;
}

function resolveConfig() {
  const envFile =
    process.argv[2] ?? process.env.AUDIT_ENV_FILE ?? DEFAULT_ENV_FILE;
  const fileValues = existsSync(envFile) ? loadEnvFile(envFile) : {};

  const uri = process.env.DB_URI ?? fileValues.DB_URI;
  const dbName = process.env.DB_NAME ?? fileValues.DB_NAME;

  if (!uri || !dbName) {
    throw new Error(
      `DB_URI/DB_NAME not found (env file: ${envFile}). ` +
        'Set them in the environment or pass an env file path.',
    );
  }

  return { uri, dbName, envFile };
}

// ------------------------------------------------- read-only wrappers
const WRITE_METHODS = new Set([
  'insertOne',
  'insertMany',
  'updateOne',
  'updateMany',
  'replaceOne',
  'deleteOne',
  'deleteMany',
  'findOneAndDelete',
  'findOneAndReplace',
  'findOneAndUpdate',
  'bulkWrite',
  'drop',
  'createIndex',
  'createIndexes',
  'dropIndex',
  'dropIndexes',
  'rename',
]);

/** Wraps a collection so any write method throws immediately. */
function readOnly(collection) {
  return new Proxy(collection, {
    get(target, prop, receiver) {
      if (WRITE_METHODS.has(prop)) {
        return () => {
          throw new Error(`read-only audit: ${String(prop)} is forbidden`);
        };
      }

      const value = Reflect.get(target, prop, receiver);

      if (prop === 'aggregate') {
        return (pipeline, options) => {
          const stages = JSON.stringify(pipeline);

          if (stages.includes('"$out"') || stages.includes('"$merge"')) {
            throw new Error('read-only audit: $out/$merge is forbidden');
          }

          return target.aggregate(pipeline, options);
        };
      }

      return typeof value === 'function' ? value.bind(target) : value;
    },
  });
}

// --------------------------------------------------------- measure kit
const OBJECT_ID_HEX = /^[0-9a-fA-F]{24}$/;

async function docCount(collection) {
  return collection.countDocuments({});
}

async function indexInventory(collection) {
  const indexes = await collection.listIndexes().toArray();

  return indexes.map((index) => ({
    name: index.name,
    key: JSON.stringify(index.key),
    unique: index.unique === true,
    expireAfterSeconds: index.expireAfterSeconds,
  }));
}

async function missingVersionCount(collection) {
  return collection.countDocuments({ __v: { $exists: false } });
}

/** Top-level field-name + BSON-type frequency, computed server-side. */
async function topLevelShape(collection) {
  return collection
    .aggregate([
      { $project: { fields: { $objectToArray: '$$ROOT' } } },
      { $unwind: '$fields' },
      {
        $group: {
          _id: { field: '$fields.k', type: { $type: '$fields.v' } },
          count: { $sum: 1 },
        },
      },
      { $sort: { '_id.field': 1, '_id.type': 1 } },
    ])
    .toArray();
}

/**
 * Field-name + type frequency inside a nested object or array element,
 * computed server-side. `segments` descends into the document; each
 * segment is `{ name, kind: 'object' | 'array' }`.
 */
async function nestedShape(collection, label, segments) {
  const pipeline = [];
  let accessor = '';

  for (const segment of segments) {
    accessor = accessor ? `${accessor}.${segment.name}` : segment.name;

    if (segment.kind === 'array') {
      pipeline.push({ $match: { [accessor]: { $type: 'array' } } });
      pipeline.push({ $unwind: `$${accessor}` });
    } else {
      pipeline.push({ $match: { [accessor]: { $type: 'object' } } });
    }
  }

  pipeline.push({ $project: { fields: { $objectToArray: `$${accessor}` } } });
  pipeline.push({ $unwind: '$fields' });
  pipeline.push({
    $group: {
      _id: {
        field: { $concat: [`${label}.`, '$fields.k'] },
        type: { $type: '$fields.v' },
      },
      count: { $sum: 1 },
    },
  });
  pipeline.push({ $sort: { '_id.field': 1, '_id.type': 1 } });

  return collection.aggregate(pipeline).toArray();
}

/** Element-type frequency for a scalar-array field. */
async function scalarArrayShape(collection, path) {
  return collection
    .aggregate([
      { $match: { [path]: { $type: 'array' } } },
      { $unwind: `$${path}` },
      {
        $group: {
          _id: { field: `${path}[]`, type: { $type: `$${path}` } },
          count: { $sum: 1 },
        },
      },
      { $sort: { '_id.type': 1 } },
    ])
    .toArray();
}

/**
 * Duplicate groups for a natural key. Returns the number of groups with
 * more than one document and the total documents in those groups.
 */
async function duplicateGroups(collection, keyExpression, preMatch = {}) {
  const rows = await collection
    .aggregate([
      { $match: preMatch },
      { $group: { _id: keyExpression, count: { $sum: 1 } } },
      { $match: { count: { $gt: 1 } } },
      {
        $group: {
          _id: null,
          groups: { $sum: 1 },
          documents: { $sum: '$count' },
        },
      },
    ])
    .toArray();

  return rows[0] ?? { groups: 0, documents: 0 };
}

/** Counts documents whose reference has no target document. */
async function danglingRefs(collection, refField, targetCollection) {
  const rows = await collection
    .aggregate([
      { $match: { [refField]: { $exists: true, $ne: null } } },
      {
        $lookup: {
          from: targetCollection,
          localField: refField,
          foreignField: '_id',
          as: '__auditMatch',
        },
      },
      { $match: { __auditMatch: { $size: 0 } } },
      { $count: 'dangling' },
    ])
    .toArray();

  return rows[0]?.dangling ?? 0;
}

/** Counts documents where a reference field is present (non-null). */
async function refsPresent(collection, refField) {
  return collection.countDocuments({
    [refField]: { $exists: true, $ne: null },
  });
}

/** Closed-enum value distribution (safe aggregate, never PII). */
async function enumDistribution(collection, field, unwindPath = null) {
  const stages = [];

  if (unwindPath) {
    stages.push({ $unwind: `$${unwindPath}` });
  }

  stages.push({ $group: { _id: `$${field}`, count: { $sum: 1 } } });
  stages.push({ $sort: { _id: 1 } });

  const rows = await collection.aggregate(stages).toArray();

  return Object.fromEntries(
    rows.map((row) => [row._id === null ? '(absent)' : String(row._id), row.count]),
  );
}

// --------------------------------------------- malformed question scan
/**
 * Checks one embedded question object and returns issue codes. Only
 * counts are reported; question content never leaves this function.
 */
function checkQuestion(question) {
  const issues = [];

  if (!question || typeof question !== 'object') {
    return ['question-not-object'];
  }

  if (typeof question.id !== 'string' || question.id.length === 0) {
    issues.push('missing-id');
  }

  const type = question.type;

  if (type !== 'mcq' && type !== 'written') {
    issues.push('bad-type');
  }

  if (typeof question.prompt !== 'string' || question.prompt.length === 0) {
    issues.push('missing-prompt');
  }

  if (!Array.isArray(question.targetConcepts) || question.targetConcepts.length === 0) {
    issues.push('missing-target-concepts');
  }

  const feedback = question.feedback;

  if (
    !feedback ||
    typeof feedback !== 'object' ||
    typeof feedback.correct !== 'string' ||
    typeof feedback.incorrect !== 'string'
  ) {
    issues.push('missing-feedback');
  }

  const rubrics = question.rubrics;

  if (!rubrics || typeof rubrics !== 'object') {
    issues.push('missing-rubrics');
  } else {
    if (!Array.isArray(rubrics.keyPoints) || rubrics.keyPoints.length === 0) {
      issues.push('rubrics-missing-key-points');
    }
    if (
      !Array.isArray(rubrics.misconceptions) ||
      rubrics.misconceptions.length === 0
    ) {
      issues.push('rubrics-missing-misconceptions');
    }
  }

  if (type === 'mcq') {
    const options = question.options;

    if (!Array.isArray(options) || options.length === 0) {
      issues.push('mcq-missing-options');
    } else {
      const optionIds = new Set();

      for (const option of options) {
        if (
          !option ||
          typeof option !== 'object' ||
          typeof option.id !== 'string' ||
          option.id.length === 0
        ) {
          issues.push('mcq-option-missing-id');
          continue;
        }
        if (optionIds.has(option.id)) {
          issues.push('mcq-duplicate-option-id');
        }
        optionIds.add(option.id);
        if (typeof option.text !== 'string' || option.text.length === 0) {
          issues.push('mcq-option-missing-text');
        }
      }

      if (
        typeof question.correctOptionId !== 'string' ||
        question.correctOptionId.length === 0
      ) {
        issues.push('mcq-missing-correct-option-id');
      } else if (!optionIds.has(question.correctOptionId)) {
        issues.push('mcq-correct-option-id-not-in-options');
      }
    }
  }

  if (type === 'written') {
    if (question.options !== undefined) {
      issues.push('written-has-options');
    }
    if (question.correctOptionId !== undefined) {
      issues.push('written-has-correct-option-id');
    }
  }

  return issues;
}

/**
 * Runs checkQuestion over every embedded question in the given docs and
 * returns `{ questionsScanned, setsWithIssues, issueCounts }`.
 */
function scanEmbeddedQuestions(documents, pickQuestions) {
  const issueCounts = {};
  let questionsScanned = 0;
  let setsWithIssues = 0;

  for (const document of documents) {
    const questions = pickQuestions(document);
    const seenIds = new Set();
    let setHasIssue = false;

    for (const question of questions) {
      questionsScanned += 1;

      if (question && typeof question === 'object' && question.id) {
        if (seenIds.has(question.id)) {
          issueCounts['duplicate-question-id-in-set'] =
            (issueCounts['duplicate-question-id-in-set'] ?? 0) + 1;
          setHasIssue = true;
        }
        seenIds.add(question.id);
      }

      for (const issue of checkQuestion(question)) {
        issueCounts[issue] = (issueCounts[issue] ?? 0) + 1;
        setHasIssue = true;
      }
    }

    if (setHasIssue) {
      setsWithIssues += 1;
    }
  }

  return { questionsScanned, setsWithIssues, issueCounts };
}

// ------------------------------------------------------------ audit spec
const QUESTION_ELEMENT_SEGMENTS = [{ name: 'questions', kind: 'array' }];

const COLLECTIONS = {
  users: {
    async measures(ctx, out) {
      out.duplicates['email (exact)'] = await duplicateGroups(ctx.c, {
        email: '$email',
      });
      out.duplicates['username (case-insensitive)'] = await duplicateGroups(
        ctx.c,
        { username: { $toLower: '$username' } },
      );
      out.enums.role = await enumDistribution(ctx.c, 'role');
    },
  },

  authSessions: {
    async measures(ctx, out) {
      out.duplicates['sessions per user (>1 total)'] = await duplicateGroups(
        ctx.c,
        { userId: '$userId' },
      );
      out.duplicates['active sessions per user (>1)'] = await duplicateGroups(
        ctx.c,
        { userId: '$userId' },
        { revokedAt: { $exists: false }, expiresAt: { $gt: ctx.now } },
      );
      out.counts['revoked'] = await ctx.c.countDocuments({
        revokedAt: { $exists: true },
      });
      out.counts['expired (by expiresAt)'] = await ctx.c.countDocuments({
        expiresAt: { $lte: ctx.now },
      });
      out.dangling['userId -> users'] = await danglingRefs(
        ctx.c,
        'userId',
        'users',
      );
    },
  },

  categories: {
    async measures(ctx, out) {
      out.duplicates['slug'] = await duplicateGroups(ctx.c, { slug: '$slug' });
    },
  },

  topics: {
    async measures(ctx, out) {
      out.duplicates['slug'] = await duplicateGroups(ctx.c, { slug: '$slug' });
      out.dangling['category -> categories (raw lookup)'] = await danglingRefs(
        ctx.c,
        'category',
        'categories',
      );
      out.counts['tags outside 1..2'] = await ctx.c.countDocuments({
        $or: [
          { tags: { $exists: false } },
          { tags: { $size: 0 } },
          { 'tags.2': { $exists: true } },
        ],
      });

      // Legacy: some category refs are stored as strings, not ObjectIds.
      // Distinguish type mismatches from truly dangling references.
      const stringRefs = await ctx.c
        .find({ category: { $type: 'string' } }, { projection: { category: 1 } })
        .toArray();
      const resolvable = stringRefs
        .map((doc) => doc.category)
        .filter((value) => OBJECT_ID_HEX.test(value))
        .map((value) => new ObjectId(value));
      const resolved =
        resolvable.length === 0
          ? 0
          : await readOnly(ctx.db.collection('categories')).countDocuments({
              _id: { $in: resolvable },
            });

      out.counts['category refs stored as string'] = stringRefs.length;
      out.counts['string category refs resolving to a category'] = resolved;
      out.counts['string category refs truly dangling'] =
        stringRefs.length - resolved;
    },
  },

  questionSets: {
    shapes: [
      { label: 'questions', segments: QUESTION_ELEMENT_SEGMENTS },
      {
        label: 'questions.feedback',
        segments: [
          { name: 'questions', kind: 'array' },
          { name: 'feedback', kind: 'object' },
        ],
      },
      {
        label: 'questions.rubrics',
        segments: [
          { name: 'questions', kind: 'array' },
          { name: 'rubrics', kind: 'object' },
        ],
      },
      {
        label: 'questions.options',
        segments: [
          { name: 'questions', kind: 'array' },
          { name: 'options', kind: 'array' },
        ],
      },
    ],
    scalarArrays: ['questions.targetConcepts'],
    async measures(ctx, out) {
      out.duplicates['topic + level + setType'] = await duplicateGroups(
        ctx.c,
        { topic: '$topic', level: '$level', setType: '$setType' },
      );
      out.dangling['topic -> topics'] = await danglingRefs(
        ctx.c,
        'topic',
        'topics',
      );
      out.enums.setType = await enumDistribution(ctx.c, 'setType');
      out.enums['questions.type'] = await enumDistribution(
        ctx.c,
        'questions.type',
        'questions',
      );

      const documents = await ctx.c
        .find({}, { projection: { questions: 1 } })
        .toArray();
      const scan = scanEmbeddedQuestions(documents, (doc) =>
        Array.isArray(doc.questions) ? doc.questions : [],
      );

      out.counts['embedded questions scanned'] = scan.questionsScanned;
      out.counts['question sets with malformed questions'] =
        scan.setsWithIssues;
      out.malformed = scan.issueCounts;
    },
  },

  sessions: {
    shapes: [
      {
        label: 'overallEvaluation',
        segments: [{ name: 'overallEvaluation', kind: 'object' }],
      },
    ],
    async measures(ctx, out) {
      out.duplicates['student + topic (total)'] = await duplicateGroups(
        ctx.c,
        { student: '$student', topic: '$topic' },
      );
      out.duplicates['student + topic (active)'] = await duplicateGroups(
        ctx.c,
        { student: '$student', topic: '$topic' },
        { status: 'active' },
      );
      out.dangling['student -> users'] = await danglingRefs(
        ctx.c,
        'student',
        'users',
      );
      out.dangling['topic -> topics'] = await danglingRefs(
        ctx.c,
        'topic',
        'topics',
      );
      out.enums.status = await enumDistribution(ctx.c, 'status');
      out.counts['with overallEvaluation'] = await ctx.c.countDocuments({
        overallEvaluation: { $exists: true, $ne: null },
      });
    },
  },

  liveSessions: {
    shapes: [
      {
        label: 'overallEvaluation',
        segments: [{ name: 'overallEvaluation', kind: 'object' }],
      },
    ],
    async measures(ctx, out) {
      out.duplicates['student + topic (total)'] = await duplicateGroups(
        ctx.c,
        { student: '$student', topic: '$topic' },
      );
      out.duplicates['student + topic (active)'] = await duplicateGroups(
        ctx.c,
        { student: '$student', topic: '$topic' },
        { status: 'active' },
      );
      out.dangling['student -> users'] = await danglingRefs(
        ctx.c,
        'student',
        'users',
      );
      out.dangling['topic -> topics'] = await danglingRefs(
        ctx.c,
        'topic',
        'topics',
      );
      out.enums.status = await enumDistribution(ctx.c, 'status');
      out.counts['with overallEvaluation'] = await ctx.c.countDocuments({
        overallEvaluation: { $exists: true, $ne: null },
      });
    },
  },

  liveQuestions: {
    shapes: [
      { label: 'question', segments: [{ name: 'question', kind: 'object' }] },
      {
        label: 'question.feedback',
        segments: [
          { name: 'question', kind: 'object' },
          { name: 'feedback', kind: 'object' },
        ],
      },
      {
        label: 'question.rubrics',
        segments: [
          { name: 'question', kind: 'object' },
          { name: 'rubrics', kind: 'object' },
        ],
      },
      {
        label: 'question.options',
        segments: [
          { name: 'question', kind: 'object' },
          { name: 'options', kind: 'array' },
        ],
      },
      { label: 'answer', segments: [{ name: 'answer', kind: 'object' }] },
    ],
    async measures(ctx, out) {
      out.duplicates['session + level + questionNumber (all)'] =
        await duplicateGroups(ctx.c, {
          liveSession: '$liveSession',
          level: '$level',
          questionNumber: '$questionNumber',
        });
      out.duplicates['session + level + questionNumber (pending)'] =
        await duplicateGroups(
          ctx.c,
          {
            liveSession: '$liveSession',
            level: '$level',
            questionNumber: '$questionNumber',
          },
          { status: 'pending' },
        );
      out.duplicates['session + level + questionNumber (not rejected)'] =
        await duplicateGroups(
          ctx.c,
          {
            liveSession: '$liveSession',
            level: '$level',
            questionNumber: '$questionNumber',
          },
          { status: { $ne: 'rejected' } },
        );
      out.dangling['liveSession -> liveSessions'] = await danglingRefs(
        ctx.c,
        'liveSession',
        'liveSessions',
      );
      out.enums.status = await enumDistribution(ctx.c, 'status');
      out.enums['question.type'] = await enumDistribution(
        ctx.c,
        'question.type',
      );
      out.enums['answer.evaluatedBy'] = await enumDistribution(
        ctx.c,
        'answer.evaluatedBy',
      );
      out.counts['with answer'] = await refsPresent(ctx.c, 'answer');

      const documents = await ctx.c
        .find({}, { projection: { question: 1 } })
        .toArray();
      const scan = scanEmbeddedQuestions(documents, (doc) =>
        doc.question ? [doc.question] : [],
      );

      out.counts['embedded questions scanned'] = scan.questionsScanned;
      out.counts['live questions malformed'] = scan.setsWithIssues;
      out.malformed = scan.issueCounts;
    },
  },

  setAttempts: {
    shapes: [
      { label: 'answers', segments: [{ name: 'answers', kind: 'array' }] },
    ],
    scalarArrays: ['answers.targetConcepts', 'answers.strengths', 'answers.weaknesses'],
    async measures(ctx, out) {
      out.duplicates['session + questionSet'] = await duplicateGroups(
        ctx.c,
        { session: '$session', questionSet: '$questionSet' },
        { session: { $exists: true, $ne: null } },
      );
      out.duplicates['liveSession + questionSet'] = await duplicateGroups(
        ctx.c,
        { liveSession: '$liveSession', questionSet: '$questionSet' },
        { liveSession: { $exists: true, $ne: null } },
      );
      out.duplicates['session + level'] = await duplicateGroups(
        ctx.c,
        { session: '$session', level: '$level' },
        { session: { $exists: true, $ne: null } },
      );
      out.duplicates['liveSession + level'] = await duplicateGroups(
        ctx.c,
        { liveSession: '$liveSession', level: '$level' },
        { liveSession: { $exists: true, $ne: null } },
      );
      out.dangling['user -> users'] = await danglingRefs(ctx.c, 'user', 'users');
      out.dangling['session -> sessions (when set)'] = await danglingRefs(
        ctx.c,
        'session',
        'sessions',
      );
      out.dangling['liveSession -> liveSessions (when set)'] =
        await danglingRefs(ctx.c, 'liveSession', 'liveSessions');
      out.dangling['topic -> topics'] = await danglingRefs(
        ctx.c,
        'topic',
        'topics',
      );
      out.dangling['questionSet -> questionSets'] = await danglingRefs(
        ctx.c,
        'questionSet',
        'questionSets',
      );
      out.counts['session and liveSession both set'] =
        await ctx.c.countDocuments({
          session: { $exists: true, $ne: null },
          liveSession: { $exists: true, $ne: null },
        });
      out.counts['neither session nor liveSession set'] =
        await ctx.c.countDocuments({
          session: { $in: [null] },
          liveSession: { $in: [null] },
        });
      out.counts['passed'] = await ctx.c.countDocuments({ passed: true });
      out.counts['failed'] = await ctx.c.countDocuments({ passed: false });
      out.enums['answers.evaluatedBy'] = await enumDistribution(
        ctx.c,
        'answers.evaluatedBy',
        'answers',
      );
    },
  },

  sessionEvaluations: {
    scalarArrays: ['attemptIds', 'strengths', 'weaknesses', 'recommendations'],
    async measures(ctx, out) {
      out.duplicates['session + fromLevel + toLevel'] = await duplicateGroups(
        ctx.c,
        {
          session: '$session',
          fromLevel: '$fromLevel',
          toLevel: '$toLevel',
        },
        { session: { $exists: true, $ne: null } },
      );
      out.duplicates['liveSession + fromLevel + toLevel'] =
        await duplicateGroups(
          ctx.c,
          {
            liveSession: '$liveSession',
            fromLevel: '$fromLevel',
            toLevel: '$toLevel',
          },
          { liveSession: { $exists: true, $ne: null } },
        );
      out.dangling['student -> users'] = await danglingRefs(
        ctx.c,
        'student',
        'users',
      );
      out.dangling['session -> sessions (when set)'] = await danglingRefs(
        ctx.c,
        'session',
        'sessions',
      );
      out.dangling['liveSession -> liveSessions (when set)'] =
        await danglingRefs(ctx.c, 'liveSession', 'liveSessions');
      out.dangling['topic -> topics'] = await danglingRefs(
        ctx.c,
        'topic',
        'topics',
      );
      out.counts['session and liveSession both set'] =
        await ctx.c.countDocuments({
          session: { $exists: true, $ne: null },
          liveSession: { $exists: true, $ne: null },
        });
      out.counts['neither session nor liveSession set'] =
        await ctx.c.countDocuments({
          session: { $in: [null] },
          liveSession: { $in: [null] },
        });

      // attemptIds are stored as plain strings, not ObjectIds.
      const referencedIds = await ctx.c.distinct('attemptIds');
      const invalidFormat = referencedIds.filter(
        (id) => typeof id !== 'string' || !OBJECT_ID_HEX.test(id),
      ).length;
      const validIds = referencedIds.filter(
        (id) => typeof id === 'string' && OBJECT_ID_HEX.test(id),
      );
      const existing =
        validIds.length === 0
          ? 0
          : await readOnly(ctx.db.collection('setAttempts')).countDocuments({
              _id: { $in: validIds.map((id) => new ObjectId(id)) },
            });

      out.counts['distinct attemptIds referenced'] = referencedIds.length;
      out.counts['attemptIds with invalid ObjectId format'] = invalidFormat;
      out.counts['dangling attemptIds (distinct)'] =
        validIds.length - existing;
    },
  },

  aiLogs: {
    async measures(ctx, out) {
      out.dangling['liveQuestion -> liveQuestions (when set)'] =
        await danglingRefs(ctx.c, 'liveQuestion', 'liveQuestions');
      out.counts['unlinked (no liveQuestion)'] = await ctx.c.countDocuments({
        liveQuestion: { $in: [null] },
      });
      out.enums.operation = await enumDistribution(ctx.c, 'operation');
    },
  },
};

// ------------------------------------------------------------ report gen
function renderShapeTable(rows) {
  if (rows.length === 0) {
    return '_none_\n';
  }

  const lines = ['| field | BSON type | documents |', '| --- | --- | ---: |'];

  for (const row of rows) {
    lines.push(`| ${row._id.field} | ${row._id.type} | ${row.count} |`);
  }

  return `${lines.join('\n')}\n`;
}

function renderDuplicates(duplicates) {
  const entries = Object.entries(duplicates);

  if (entries.length === 0) {
    return '_none_\n';
  }

  const lines = ['| natural key | duplicate groups | documents in groups |', '| --- | ---: | ---: |'];

  for (const [key, value] of entries) {
    lines.push(`| ${key} | ${value.groups} | ${value.documents} |`);
  }

  return `${lines.join('\n')}\n`;
}

function renderCounts(counts) {
  const entries = Object.entries(counts);

  if (entries.length === 0) {
    return '_none_\n';
  }

  return `${entries.map(([key, value]) => `- ${key}: **${value}**`).join('\n')}\n`;
}

function renderEnums(enums) {
  const entries = Object.entries(enums);

  if (entries.length === 0) {
    return '_none_\n';
  }

  return `${entries
    .map(
      ([key, distribution]) =>
        `- ${key}: ${Object.entries(distribution)
          .map(([value, count]) => `\`${value}\` × ${count}`)
          .join(', ')}`,
    )
    .join('\n')}\n`;
}

// ------------------------------------------------------------------ main
async function main() {
  const { uri, dbName, envFile } = resolveConfig();
  const client = new MongoClient(uri, {
    appName: 'synaptic-contract-audit-readonly',
    readPreference: 'secondaryPreferred',
  });

  await client.connect();

  try {
    const db = client.db(dbName);
    const collectionNames = (
      await db.listCollections({}, { nameOnly: true }).toArray()
    )
      .map((info) => info.name)
      .sort();
    const now = new Date();
    const sections = [];

    for (const [name, spec] of Object.entries(COLLECTIONS)) {
      if (!collectionNames.includes(name)) {
        sections.push({ name, missing: true });
        continue;
      }

      const collection = readOnly(db.collection(name));
      const ctx = { c: collection, db, now };
      const out = {
        counts: {},
        duplicates: {},
        dangling: {},
        enums: {},
        malformed: {},
        shapeRows: await topLevelShape(collection),
      };

      out.counts['documents'] = await docCount(collection);
      out.counts['documents missing __v'] = await missingVersionCount(
        collection,
      );
      out.indexes = await indexInventory(collection);

      for (const shape of spec.shapes ?? []) {
        out.shapeRows.push(
          ...(await nestedShape(collection, shape.label, shape.segments)),
        );
      }

      for (const path of spec.scalarArrays ?? []) {
        out.shapeRows.push(await scalarArrayShape(collection, path));
      }

      out.shapeRows = out.shapeRows
        .flat()
        .sort((a, b) =>
          `${a._id.field}${a._id.type}`.localeCompare(
            `${b._id.field}${b._id.type}`,
          ),
        );

      await spec.measures(ctx, out);
      sections.push({ name, out });
    }

    const report = renderReport(dbName, envFile, now, collectionNames, sections);
    const reportPath = join(scriptDir, 'report.md');

    writeFileSync(reportPath, report);
    console.log(`Audit complete: ${sections.length} collections -> ${reportPath}`);
  } finally {
    await client.close();
  }
}

function renderReport(dbName, envFile, now, collectionNames, sections) {
  const lines = [];

  lines.push('# BSON audit report');
  lines.push('');
  lines.push(`- Generated: ${now.toISOString()} (run timestamp)`);
  lines.push(`- Target: database \`${dbName}\` (read-only audit)`);
  lines.push(`- Env file: \`${basename(envFile)}\` (path elided; credentials never read into output)`);
  lines.push('- Method: countDocuments, aggregate, find (projections),');
  lines.push('  distinct, listIndexes only; no writes of any kind.');
  lines.push('- Contents: aggregate counts, field names, BSON types, and');
  lines.push('  index specs only — no user data, tokens, or content values.');
  lines.push('');
  lines.push(`Collections present (${collectionNames.length}):`);
  lines.push(collectionNames.map((name) => `\`${name}\``).join(', '));
  lines.push('');

  for (const section of sections) {
    lines.push(`## ${section.name}`);
    lines.push('');

    if (section.missing) {
      lines.push('**Collection not present in the database.**');
      lines.push('');
      continue;
    }

    const { out } = section;

    lines.push('### Counts');
    lines.push('');
    lines.push(renderCounts(out.counts));
    lines.push('### Indexes');
    lines.push('');
    lines.push('| name | key | unique | expireAfterSeconds |');
    lines.push('| --- | --- | --- | --- |');
    for (const index of out.indexes) {
      lines.push(
        `| ${index.name} | \`${index.key}\` | ${index.unique} | ` +
          `${index.expireAfterSeconds ?? ''} |`,
      );
    }
    lines.push('');
    lines.push('### Duplicate natural keys');
    lines.push('');
    lines.push(renderDuplicates(out.duplicates));
    lines.push('### Dangling references');
    lines.push('');
    const danglingEntries = Object.entries(out.dangling);
    lines.push(
      danglingEntries.length === 0
        ? '_none_\n'
        : `${danglingEntries.map(([key, value]) => `- ${key}: **${value}**`).join('\n')}\n`,
    );
    lines.push('### Enum distributions');
    lines.push('');
    lines.push(renderEnums(out.enums));
    lines.push('### Malformed embedded questions');
    lines.push('');
    const malformedEntries = Object.entries(out.malformed);
    lines.push(
      malformedEntries.length === 0
        ? '_none found_\n'
        : `${malformedEntries
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([key, value]) => `- ${key}: **${value}**`)
            .join('\n')}\n`,
    );
    lines.push('### Field shape (name + BSON type frequency)');
    lines.push('');
    lines.push(renderShapeTable(out.shapeRows));
  }

  lines.push('## Findings for the Go BSON mappers');
  lines.push('');
  lines.push('Derived from the measured values above (snapshot-specific).');
  lines.push('');
  const findings = collectFindings(sections, collectionNames);

  if (findings.length === 0) {
    lines.push('- No legacy anomalies detected in this snapshot.');
  } else {
    for (const finding of findings) {
      lines.push(`- ${finding}`);
    }
  }

  lines.push('');
  lines.push('Structural facts independent of this snapshot:');
  lines.push('');
  lines.push('- `sessionEvaluations.attemptIds` is stored as an array of');
  lines.push('  **strings**, not ObjectIds (see schema and shape table).');
  lines.push('- Optional refs (`session`/`liveSession` on setAttempts and');
  lines.push('  sessionEvaluations, `liveQuestion` on aiLogs) may be absent');
  lines.push('  or null; mappers must treat them as omitempty pointers.');
  lines.push('- Embedded question shapes are polymorphic on `type`; malformed');
  lines.push('  counts above describe exactly which legacy deviations exist.');
  lines.push('');

  return `${lines.join('\n')}\n`;
}

/** Derives snapshot-specific anomaly findings from measured sections. */
function collectFindings(sections, collectionNames) {
  const findings = [];
  const escapeDots = (value) => value.replace(/\./g, '\\.');
  const byName = Object.fromEntries(
    sections.filter((section) => section.out).map((s) => [s.name, s.out]),
  );
  const shapeHas = (name, fieldPattern) =>
    (byName[name]?.shapeRows ?? []).some((row) =>
      row._id.field.match(fieldPattern),
    );
  const enumKeys = (name, field) => Object.keys(byName[name]?.enums[field] ?? {});

  const legacyCollections = collectionNames.filter(
    (name) => !(name in COLLECTIONS),
  );

  if (legacyCollections.length > 0) {
    findings.push(
      `Collections present but NOT part of the current 11-collection ` +
        `schema (not audited): ${legacyCollections.map((n) => `\`${n}\``).join(', ')}.`,
    );
  }

  const missingV = Object.entries(byName)
    .filter(([, out]) => out.counts['documents missing __v'] > 0)
    .map(([name, out]) => `\`${name}\` (${out.counts['documents missing __v']})`);

  if (missingV.length > 0) {
    findings.push(`Documents missing \`__v\`: ${missingV.join(', ')}.`);
  }

  const setTypes = enumKeys('questionSets', 'setType').filter(
    (value) => !['regular', 'live'].includes(value),
  );

  if (setTypes.length > 0) {
    findings.push(
      `\`questionSets.setType\` contains values outside the current enum ` +
        `(\`regular\`/\`live\`): ${setTypes.map((v) => `\`${v}\``).join(', ')}.`,
    );
  }

  const liveStatuses = enumKeys('liveQuestions', 'status').filter(
    (value) => !['pending', 'rejected', 'passed', 'failed'].includes(value),
  );

  if (liveStatuses.length > 0) {
    findings.push(
      `\`liveQuestions.status\` contains values outside the current enum: ` +
        `${liveStatuses.map((v) => `\`${v}\``).join(', ')}.`,
    );
  }

  if (shapeHas('topics', /^category$/)) {
    const stringRows = (byName.topics?.shapeRows ?? []).find(
      (row) => row._id.field === 'category' && row._id.type === 'string',
    );

    if (stringRows) {
      findings.push(
        `\`topics.category\` is stored as a **string** (not ObjectId) in ` +
          `${stringRows.count} documents — see the string-ref breakdown in ` +
          'the topics section.',
      );
    }
  }

  const questionExtras = [
    ['questions.hints', '`questions[].hints`'],
    ['questions.feedback.sampleAnswer', '`questions[].feedback.sampleAnswer`'],
    [
      'questions.feedback.evaluationGuidance',
      '`questions[].feedback.evaluationGuidance`',
    ],
  ]
    .filter(([field]) =>
      shapeHas('questionSets', new RegExp(`^${escapeDots(field)}$`)),
    )
    .map(([, label]) => label);

  if (questionExtras.length > 0) {
    findings.push(
      `Embedded questions carry fields absent from the current DTO: ` +
        `${questionExtras.join(', ')}.`,
    );
  }

  if (
    shapeHas('questionSets', /^questions\.options$/) &&
    (byName.questionSets?.shapeRows ?? []).some(
      (row) => row._id.field === 'questions.options' && row._id.type === 'null',
    )
  ) {
    findings.push(
      'Some written questions carry `options`/`correctOptionId` with ' +
        '**null** values (not absent fields) — treat both as omitempty.',
    );
  }

  const singularVariants = [
    ['setAttempts', 'answers.strength'],
    ['setAttempts', 'answers.weakness'],
    ['setAttempts', 'strength'],
    ['setAttempts', 'weakness'],
    ['sessionEvaluations', 'stength'],
    ['sessionEvaluations', 'recommendation'],
    ['sessionEvaluations', 'weakness'],
  ]
    .filter(([name, field]) =>
      shapeHas(name, new RegExp(`^${escapeDots(field)}$`)),
    )
    .map(([name, field]) => `${name}.${field}`);

  if (singularVariants.length > 0) {
    findings.push(
      'Singular/typo field variants exist alongside the plural forms: ' +
        `${singularVariants.map((v) => `\`${v}\``).join(', ')}.`,
    );
  }

  const mixedNumeric = Object.entries(byName).flatMap(([name, out]) => {
    const byField = {};

    for (const row of out.shapeRows) {
      byField[row._id.field] ??= new Set();
      byField[row._id.field].add(row._id.type);
    }

    return Object.entries(byField)
      .filter(
        ([field, types]) =>
          /score/i.test(field) && types.has('int') && types.has('double'),
      )
      .map(([field]) => `\`${name}.${field}\``);
  });

  if (mixedNumeric.length > 0) {
    findings.push(
      `Score fields mix **int** and **double** BSON types: ` +
        `${mixedNumeric.join(', ')}.`,
    );
  }

  const duplicateSummary = Object.entries(byName).flatMap(([name, out]) =>
    Object.entries(out.duplicates)
      .filter(([, value]) => value.groups > 0)
      .map(
        ([key, value]) =>
          `\`${name}\` ${key} (${value.groups} groups / ${value.documents} docs)`,
      ),
  );

  if (duplicateSummary.length > 0) {
    findings.push(`Duplicate natural keys exist: ${duplicateSummary.join('; ')}.`);
  }

  const danglingSummary = Object.entries(byName).flatMap(([name, out]) =>
    Object.entries(out.dangling)
      .filter(([, value]) => value > 0)
      .map(([key, value]) => `\`${name}\` ${key} (${value})`),
  );

  if (danglingSummary.length > 0) {
    findings.push(
      `Dangling references exist: ${danglingSummary.join('; ')}.`,
    );
  }

  return findings;
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
