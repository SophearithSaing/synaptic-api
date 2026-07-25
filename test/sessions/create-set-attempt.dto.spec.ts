import { plainToInstance } from 'class-transformer';
import { validate } from 'class-validator';
import { QuestionType } from '../../src/questions/schemas/question-set.schema';
import { CreateSetAttemptDto } from '../../src/sessions/dtos';
import { EvaluatedBy } from '../../src/sessions/schemas/set-attempt.schemas';

describe('CreateSetAttemptDto', () => {
  const mongoId = '507f1f77bcf86cd799439011';
  const validPayload = {
    user: mongoId,
    session: mongoId,
    topic: mongoId,
    questionSet: mongoId,
    level: 0,
    answers: [
      {
        id: 'ans-q1',
        questionId: 'q1',
        questionType: QuestionType.MCQ,
        answer: 'o1',
        correctAnswer: 'o1',
        score: 1,
        feedback: 'Correct.',
        targetConcepts: ['paging'],
        strengths: ['paging'],
        weaknesses: [],
        evaluatedBy: EvaluatedBy.System,
      },
    ],
    setScore: 1,
    passed: true,
    strengths: ['paging'],
    weaknesses: [],
    submittedAt: new Date('2026-01-01T00:00:00.000Z'),
    evaluatedAt: new Date('2026-01-01T00:00:00.000Z'),
  };

  it('accepts a regular session reference', async () => {
    const dto = plainToInstance(CreateSetAttemptDto, validPayload);

    await expect(validate(dto)).resolves.toHaveLength(0);
  });

  it('accepts a live session reference', async () => {
    const dto = plainToInstance(CreateSetAttemptDto, {
      ...validPayload,
      session: undefined,
      liveSession: mongoId,
    });

    await expect(validate(dto)).resolves.toHaveLength(0);
  });

  it('requires a session when liveSession is missing', async () => {
    const dto = plainToInstance(CreateSetAttemptDto, {
      ...validPayload,
      session: undefined,
    });

    const errors = await validate(dto);

    expect(errors).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ property: 'session' }),
        expect.objectContaining({ property: 'liveSession' }),
      ]),
    );
  });

  it(
    'rejects an empty provided session even when liveSession exists',
    async () => {
      const dto = plainToInstance(CreateSetAttemptDto, {
        ...validPayload,
        session: '',
        liveSession: mongoId,
      });

      const errors = await validate(dto);

      expect(errors).toEqual(
        expect.arrayContaining([
          expect.objectContaining({
            constraints: expect.objectContaining({
              isNotEmpty: expect.any(String),
            }),
            property: 'session',
          }),
        ]),
      );
    },
  );
});
