import { NotFoundException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Types } from 'mongoose';
import {
  Question,
  QuestionSetType,
  QuestionType,
} from '../../src/questions/schemas/question-set.schema';
import { SessionsService } from '../../src/sessions/sessions.service';
import {
  getLiveQuestionComposition,
  getNextLiveQuestionType,
} from '../../src/sessions/sessions.util';
import { LiveQuestionStatus } from '../../src/sessions/schemas/live-question.schema';
import { SessionStatus } from '../../src/sessions/schemas/session.schema';
import { EvaluatedBy } from '../../src/sessions/schemas/set-attempt.schemas';

describe('SessionsService', () => {
  const studentId = '507f1f77bcf86cd799439011';
  const topicId = new Types.ObjectId('507f1f77bcf86cd799439012');
  const liveSessionId = new Types.ObjectId('507f1f77bcf86cd799439013');
  const liveQuestionId = new Types.ObjectId('507f1f77bcf86cd799439014');
  const replacementQuestionId = new Types.ObjectId('507f1f77bcf86cd799439015');
  const questionSetId = new Types.ObjectId('507f1f77bcf86cd799439016');
  const attemptId = new Types.ObjectId('507f1f77bcf86cd799439017');
  const sessionId = new Types.ObjectId('507f1f77bcf86cd799439018');
  const topic = {
    _id: topicId,
    slug: 'memory-management',
    title: 'Memory Management',
    description: 'Memory concepts.',
    tags: ['systems'],
  };
  const questionOne: Question = {
    id: 'memory-management-l0-q1',
    type: QuestionType.MCQ,
    prompt: 'What does paging divide memory into?',
    options: [
      { id: 'o1', text: 'Pages' },
      { id: 'o2', text: 'Threads' },
      { id: 'o3', text: 'Registers' },
    ],
    correctOptionId: 'o1',
    targetConcepts: ['paging'],
    feedback: {
      correct: 'Correct.',
      incorrect: 'Review paging.',
    },
    rubrics: {
      keyPoints: ['Pages'],
      misconceptions: ['Paging uses registers'],
    },
  };
  const questionTwo: Question = {
    ...questionOne,
    id: 'memory-management-l0-q2',
    prompt: 'What maps pages to frames?',
    correctOptionId: 'o1',
    targetConcepts: ['page-table'],
  };
  const questionThree: Question = {
    ...questionOne,
    id: 'memory-management-l0-q3',
    prompt: 'What stores recent address translations?',
    targetConcepts: ['tlb'],
  };
  const nextLevelQuestion: Question = {
    ...questionOne,
    id: 'memory-management-l1-q1',
    prompt: 'What is virtual memory?',
    targetConcepts: ['virtual-memory'],
  };
  let service: SessionsService;
  let sessionModel: Record<string, jest.Mock>;
  let topicModel: Record<string, jest.Mock>;
  let questionSetModel: Record<string, jest.Mock>;
  let liveQuestionModel: Record<string, jest.Mock>;
  let liveSessionModel: Record<string, jest.Mock>;
  let setAttemptModel: Record<string, jest.Mock>;
  let sessionEvaluationModel: Record<string, jest.Mock>;
  let generateLiveQuestion: jest.Mock;

  beforeEach(() => {
    sessionModel = {
      create: jest.fn(),
      find: jest.fn(),
      findOne: jest.fn(),
      findOneAndDelete: jest.fn(),
      updateOne: jest.fn(),
    };
    topicModel = {
      findById: jest.fn(),
    };
    questionSetModel = {
      create: jest.fn(),
      findById: jest.fn(),
      findOne: jest.fn(),
    };
    liveQuestionModel = {
      create: jest.fn(),
      find: jest.fn(),
      findOne: jest.fn(),
      findOneAndDelete: jest.fn(),
      updateOne: jest.fn(),
    };
    liveSessionModel = {
      create: jest.fn(),
      findOne: jest.fn(),
      updateOne: jest.fn(),
    };
    setAttemptModel = {
      create: jest.fn(),
      find: jest.fn(),
    };
    sessionEvaluationModel = {
      create: jest.fn(),
      find: jest.fn(),
    };
    service = new SessionsService(
      sessionModel as never,
      topicModel as never,
      questionSetModel as never,
      liveQuestionModel as never,
      liveSessionModel as never,
      setAttemptModel as never,
      sessionEvaluationModel as never,
      { get: jest.fn() } as unknown as ConfigService,
    );
    generateLiveQuestion = jest
      .spyOn(
        service as unknown as {
          generateLiveQuestion(context: unknown): Promise<Question>;
        },
        'generateLiveQuestion',
      )
      .mockResolvedValue(nextLevelQuestion);
  });

  it('maps live question ratios at documented thresholds', () => {
    expect(getLiveQuestionComposition(0)).toEqual({ mcq: 3, written: 0 });
    expect(getLiveQuestionComposition(10)).toEqual({ mcq: 3, written: 0 });
    expect(getLiveQuestionComposition(11)).toEqual({ mcq: 2, written: 1 });
    expect(getLiveQuestionComposition(21)).toEqual({ mcq: 1, written: 2 });
    expect(getLiveQuestionComposition(31)).toEqual({ mcq: 0, written: 3 });
    expect(getNextLiveQuestionType(11, [questionOne, questionTwo])).toBe(
      QuestionType.Written,
    );
  });

  it('starts a live session with one pending generated question', async () => {
    topicModel.findById.mockReturnValue(execQuery(topic));
    liveSessionModel.create.mockResolvedValue({ _id: liveSessionId });
    liveQuestionModel.create.mockResolvedValue({ _id: liveQuestionId });
    generateLiveQuestion.mockResolvedValue(questionOne);

    const result = await service.startLiveSession(
      topicId.toString(),
      studentId,
    );

    expect(result).toEqual({
      sessionId: liveSessionId.toString(),
      questionId: liveQuestionId.toString(),
      question: questionOne,
    });
    expect(liveSessionModel.create).toHaveBeenCalledWith(
      expect.objectContaining({
        currentLevel: 0,
        status: SessionStatus.Active,
      }),
    );
    expect(liveQuestionModel.create).toHaveBeenCalledWith(
      expect.objectContaining({ status: LiveQuestionStatus.Pending }),
    );
  });

  it('continues a live session by returning the current pending question', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(
      sortableQuery({ _id: liveQuestionId, question: questionOne }),
    );

    const result = await service.continueLiveSession(
      studentId,
      liveSessionId.toString(),
    );

    expect(result.questionId).toBe(liveQuestionId.toString());
    expect(result.question).toEqual(questionOne);
    expect(generateLiveQuestion).not.toHaveBeenCalled();
  });

  it('continues a live session by generating the next required question', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(sortableQuery(null));
    liveQuestionModel.find.mockReturnValue(
      sortableQuery([
        { question: questionOne, status: LiveQuestionStatus.Accepted },
      ]),
    );
    topicModel.findById.mockReturnValue(execQuery(topic));
    liveQuestionModel.create.mockResolvedValue({ _id: replacementQuestionId });
    generateLiveQuestion.mockResolvedValue(questionTwo);

    const result = await service.continueLiveSession(
      studentId,
      liveSessionId.toString(),
    );

    expect(result.question).toEqual(questionTwo);
    expect(liveQuestionModel.create).toHaveBeenCalledWith(
      expect.objectContaining({ questionNumber: 2 }),
    );
  });

  it('rejects only pending live questions and returns a replacement', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 0 }),
    );
    liveQuestionModel.findOneAndDelete.mockReturnValue(
      execQuery({ _id: liveQuestionId, question: questionOne }),
    );
    topicModel.findById.mockReturnValue(execQuery(topic));
    liveQuestionModel.find.mockReturnValue(sortableQuery([]));
    liveQuestionModel.create.mockResolvedValue({ _id: replacementQuestionId });
    generateLiveQuestion.mockResolvedValue(questionOne);

    const result = await service.rejectLiveQuestion(
      studentId,
      liveSessionId.toString(),
      liveQuestionId.toString(),
      'Ambiguous.',
    );

    expect(result.questionId).toBe(replacementQuestionId.toString());
    expect(liveQuestionModel.findOneAndDelete).toHaveBeenCalledWith(
      expect.objectContaining({ status: LiveQuestionStatus.Pending }),
    );
  });

  it('rejects stale live question rejection requests', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, currentLevel: 0 }),
    );
    liveQuestionModel.findOneAndDelete.mockReturnValue(execQuery(null));

    await expect(
      service.rejectLiveQuestion(
        studentId,
        liveSessionId.toString(),
        liveQuestionId.toString(),
        'Duplicate.',
      ),
    ).rejects.toThrow(NotFoundException);
  });

  it('rejects duplicate live answer submissions', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(
      execQuery({ _id: liveQuestionId, question: questionOne }),
    );
    liveQuestionModel.updateOne.mockReturnValue(
      execQuery({ modifiedCount: 0 }),
    );

    await expect(
      service.submitLiveAnswer(
        studentId,
        liveSessionId.toString(),
        liveQuestionId.toString(),
        'o1',
      ),
    ).rejects.toThrow(NotFoundException);
    expect(liveQuestionModel.updateOne).toHaveBeenCalledWith(
      expect.objectContaining({ status: LiveQuestionStatus.Pending }),
      expect.any(Object),
    );
  });

  it('submits one live answer and returns the next pending question', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(
      execQuery({ _id: liveQuestionId, question: questionOne }),
    );
    liveQuestionModel.updateOne.mockReturnValue(
      execQuery({ modifiedCount: 1 }),
    );
    liveQuestionModel.find.mockReturnValue(
      sortableQuery([
        { question: questionOne, answer: createAnswer(questionOne, 'o1') },
      ]),
    );
    topicModel.findById.mockReturnValue(execQuery(topic));
    liveQuestionModel.create.mockResolvedValue({ _id: replacementQuestionId });
    generateLiveQuestion.mockResolvedValue(questionTwo);

    const result = await service.submitLiveAnswer(
      studentId,
      liveSessionId.toString(),
      liveQuestionId.toString(),
      'o1',
    );

    expect(result.answers).toHaveLength(1);
    expect(result.answers[0].evaluatedBy).toBe(EvaluatedBy.System);
    expect(result.nextQuestion?.question).toEqual(questionTwo);
  });

  it('saves completed passing live sets and generates the next level question', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(
      execQuery({ _id: liveQuestionId, question: questionThree }),
    );
    liveQuestionModel.updateOne.mockReturnValue(
      execQuery({ modifiedCount: 1 }),
    );
    liveQuestionModel.find.mockReturnValue(
      sortableQuery([
        { question: questionOne, answer: createAnswer(questionOne, 'o1') },
        { question: questionTwo, answer: createAnswer(questionTwo, 'o1') },
        { question: questionThree, answer: createAnswer(questionThree, 'o1') },
      ]),
    );
    questionSetModel.create.mockResolvedValue(createQuestionSet(0));
    setAttemptModel.create.mockResolvedValue(
      createAttempt([questionOne, questionTwo, questionThree], true),
    );
    topicModel.findById.mockReturnValue(execQuery(topic));
    liveQuestionModel.create.mockResolvedValue({ _id: replacementQuestionId });
    generateLiveQuestion.mockResolvedValue(nextLevelQuestion);
    liveSessionModel.updateOne.mockReturnValue(execQuery({ modifiedCount: 1 }));

    const result = await service.submitLiveAnswer(
      studentId,
      liveSessionId.toString(),
      liveQuestionId.toString(),
      'o1',
    );

    expect(questionSetModel.create).toHaveBeenCalledWith(
      expect.objectContaining({ setType: QuestionSetType.Live }),
    );
    expect(setAttemptModel.create).toHaveBeenCalled();
    expect(result.answers).toHaveLength(3);
    expect(result.nextQuestion?.question).toEqual(nextLevelQuestion);
  });

  it('returns completed failing live sets without a next question', async () => {
    const failedAnswer = createAnswer(questionThree, 'o2');

    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(
      execQuery({ _id: liveQuestionId, question: questionThree }),
    );
    liveQuestionModel.updateOne.mockReturnValue(
      execQuery({ modifiedCount: 1 }),
    );
    liveQuestionModel.find.mockReturnValue(
      sortableQuery([
        { question: questionOne, answer: createAnswer(questionOne, 'o1') },
        { question: questionTwo, answer: createAnswer(questionTwo, 'o1') },
        { question: questionThree, answer: failedAnswer },
      ]),
    );
    questionSetModel.create.mockResolvedValue(createQuestionSet(0));
    setAttemptModel.create.mockResolvedValue(
      createAttempt([questionOne, questionTwo, questionThree], false),
    );

    const result = await service.submitLiveAnswer(
      studentId,
      liveSessionId.toString(),
      liveQuestionId.toString(),
      'o2',
    );

    expect(result.answers).toHaveLength(3);
    expect(result.nextQuestion).toBeNull();
    expect(generateLiveQuestion).not.toHaveBeenCalled();
  });

  it('preserves regular submit-answer attempt creation flow', async () => {
    const session = { _id: sessionId, topic: topicId, currentLevel: 0 };
    const questionSet = createQuestionSet(0);
    const nextQuestionSet = createQuestionSet(1);
    const attempt = createAttempt([questionOne], true);

    sessionModel.findOne.mockReturnValue(execQuery(session));
    questionSetModel.findById.mockReturnValue(execQuery(questionSet));
    setAttemptModel.create.mockResolvedValue(attempt);
    sessionModel.updateOne.mockReturnValue(execQuery({ modifiedCount: 1 }));
    questionSetModel.findOne.mockReturnValue(execQuery(nextQuestionSet));

    const result = await service.submitAnswer(
      studentId,
      sessionId.toString(),
      questionSetId.toString(),
      [{ questionId: questionOne.id, answer: 'o1' }],
    );

    expect(setAttemptModel.create).toHaveBeenCalledWith(
      expect.objectContaining({ session: sessionId }),
    );
    expect(result.attempt.id).toBe(attemptId.toString());
    expect(result.nextQuestionSet?.id).toBe(questionSetId.toString());
  });

  const execQuery = (value: unknown): { exec: jest.Mock } => ({
    exec: jest.fn().mockResolvedValue(value),
  });

  const sortableQuery = (
    value: unknown,
  ): { exec: jest.Mock; sort: jest.Mock } => {
    const query = {
      exec: jest.fn().mockResolvedValue(value),
      sort: jest.fn(),
    };

    query.sort.mockReturnValue(query);

    return query;
  };

  const createAnswer = (question: Question, answer: string): unknown => ({
    id: `ans-${question.id}`,
    questionId: question.id,
    questionType: question.type,
    answer,
    correctAnswer: question.correctOptionId,
    score: answer === question.correctOptionId ? 1 : 0,
    feedback:
      answer === question.correctOptionId
        ? question.feedback.correct
        : question.feedback.incorrect,
    targetConcepts: question.targetConcepts,
    strengths:
      answer === question.correctOptionId ? question.targetConcepts : [],
    weaknesses:
      answer === question.correctOptionId ? [] : question.targetConcepts,
    evaluatedBy: EvaluatedBy.System,
  });

  const createQuestionSet = (level: number): unknown => ({
    _id: questionSetId,
    topic: topicId,
    setType: QuestionSetType.Live,
    level,
    questions: [questionOne],
  });

  const createAttempt = (questions: Question[], passed: boolean): unknown => ({
    _id: attemptId,
    user: new Types.ObjectId(studentId),
    session: sessionId,
    topic: topicId,
    questionSet: questionSetId,
    level: 0,
    answers: questions.map((question, index) =>
      createAnswer(
        question,
        passed || index < questions.length - 1 ? 'o1' : 'o2',
      ),
    ),
    setScore: passed ? 1 : 0.7,
    passed,
    strengths: passed ? ['paging'] : [],
    weaknesses: passed ? [] : ['tlb'],
    submittedAt: new Date('2026-01-01T00:00:00.000Z'),
    evaluatedAt: new Date('2026-01-01T00:00:00.000Z'),
    createdAt: new Date('2026-01-01T00:00:00.000Z'),
    updatedAt: new Date('2026-01-01T00:00:00.000Z'),
  });
});
