/**
 * Deterministic seed data for fixture capture.
 *
 * All seeded documents use fixed ObjectIds (`5eed…`) and fixed
 * timestamps so every run starts from an identical database state. The
 * user password hash is bcrypt-generated at seed time; it is
 * deterministic-input but volatile-output and never appears in fixtures
 * (database only).
 */
import bcrypt from 'bcrypt';
import mongoose from 'mongoose';

const { ObjectId } = mongoose.Types;

/** Converts fixed hex strings to ObjectIds for native-driver inserts. */
function objectIds(document, fields) {
  const converted = { ...document };

  for (const field of fields) {
    converted[field] = new ObjectId(document[field]);
  }

  return converted;
}

export const SEED_IDS = {
  userStudent: '5eed00000000000000000001',
  userAdmin: '5eed00000000000000000002',
  category: '5eed00000000000000000011',
  topic: '5eed00000000000000000021',
  topicEmpty: '5eed00000000000000000022',
  questionSetL0: '5eed00000000000000000031',
  questionSetL1: '5eed00000000000000000032',
  /** Valid ObjectId that never exists in the database (404 cases). */
  unknown: '5eed00000000000000000099',
};

export const SEED_PASSWORD = 'Password1';

const SEED_DATE = new Date('2026-01-01T00:00:00.000Z');

function mcqQuestion(id, prompt, correctOptionId) {
  return {
    id,
    type: 'mcq',
    prompt,
    options: [
      { id: 'o1', text: `${id} option one` },
      { id: 'o2', text: `${id} option two` },
      { id: 'o3', text: `${id} option three` },
    ],
    correctOptionId,
    targetConcepts: ['binary-addition'],
    feedback: {
      correct: 'Correct feedback.',
      incorrect: 'Incorrect feedback.',
    },
    rubrics: {
      keyPoints: ['Key point.'],
      misconceptions: ['Misconception.'],
    },
  };
}

/** Returns the seed documents grouped by collection name. */
export function buildSeedDocuments() {
  const passwordHash = bcrypt.hashSync(SEED_PASSWORD, 10);

  return {
    users: [
      objectIds(
        {
          _id: SEED_IDS.userStudent,
          username: 'student',
          email: 'student@example.com',
          password: passwordHash,
          role: 'user',
          createdAt: SEED_DATE,
          updatedAt: SEED_DATE,
          __v: 0,
        },
        ['_id'],
      ),
      objectIds(
        {
          _id: SEED_IDS.userAdmin,
          username: 'admin',
          email: 'admin@example.com',
          password: passwordHash,
          role: 'admin',
          createdAt: SEED_DATE,
          updatedAt: SEED_DATE,
          __v: 0,
        },
        ['_id'],
      ),
    ],
    categories: [
      objectIds(
        {
          _id: SEED_IDS.category,
          title: 'Core Concepts',
          slug: 'core-concepts',
          description: 'Foundational computing theory.',
          icon: 'cpu',
          createdAt: SEED_DATE,
          updatedAt: SEED_DATE,
          __v: 0,
        },
        ['_id'],
      ),
    ],
    topics: [
      objectIds(
        {
          _id: SEED_IDS.topic,
          title: 'Binary Basics',
          slug: 'binary-basics',
          description: 'Binary numbers and arithmetic.',
          icon: 'binary',
          tags: ['binary', 'arithmetic'],
          category: SEED_IDS.category,
          createdAt: SEED_DATE,
          updatedAt: SEED_DATE,
          __v: 0,
        },
        ['_id', 'category'],
      ),
      objectIds(
        {
          _id: SEED_IDS.topicEmpty,
          title: 'Empty Topic',
          slug: 'empty-topic',
          description: 'Topic without question sets.',
          icon: 'empty',
          tags: ['misc'],
          category: SEED_IDS.category,
          createdAt: SEED_DATE,
          updatedAt: SEED_DATE,
          __v: 0,
        },
        ['_id', 'category'],
      ),
    ],
    questionSets: [
      objectIds(
        {
          _id: SEED_IDS.questionSetL0,
          topic: SEED_IDS.topic,
          setType: 'regular',
          level: 0,
          questions: [
            mcqQuestion('seed-l0-q1', 'What is 1 + 1 in binary?', 'o1'),
            mcqQuestion('seed-l0-q2', 'What is 10 + 1 in binary?', 'o2'),
          ],
          createdAt: SEED_DATE,
          updatedAt: SEED_DATE,
          __v: 0,
        },
        ['_id', 'topic'],
      ),
      objectIds(
        {
          _id: SEED_IDS.questionSetL1,
          topic: SEED_IDS.topic,
          setType: 'regular',
          level: 1,
          questions: [
            {
              id: 'seed-l1-q1',
              type: 'written',
              prompt: "Explain two's complement.",
              targetConcepts: ['twos-complement'],
              feedback: {
                correct: 'Correct feedback.',
                incorrect: 'Incorrect feedback.',
              },
              rubrics: {
                keyPoints: ['Invert bits.', 'Add one.'],
                misconceptions: ['Sign bit only.'],
              },
            },
          ],
          createdAt: SEED_DATE,
          updatedAt: SEED_DATE,
          __v: 0,
        },
        ['_id', 'topic'],
      ),
    ],
  };
}

/**
 * Drops the database, recreates schema indexes (dropping the database
 * also drops the indexes that register/login duplicate detection relies
 * on), and inserts the seed documents.
 */
export async function resetAndSeed(connection) {
  await connection.dropDatabase();

  for (const model of Object.values(connection.models)) {
    await model.ensureIndexes();
  }

  const documents = buildSeedDocuments();

  for (const [collection, docs] of Object.entries(documents)) {
    await connection.collection(collection).insertMany(docs);
  }
}
