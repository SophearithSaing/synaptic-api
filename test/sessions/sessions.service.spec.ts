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
  let generateLiveQuestion: jest.SpyInstance<
    Promise<Question>,
    [context: unknown]
  >;

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
      find: jest.fn(),
      findOneAndDelete: jest.fn(),
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

  it('fetches in-progress live sessions for the authenticated user', async () => {
    const query = populateSortableQuery([
      {
        _id: liveSessionId,
        student: new Types.ObjectId(studentId),
        topic,
        currentLevel: 0,
        status: SessionStatus.Active,
      },
    ]);

    liveSessionModel.find.mockReturnValue(query);

    const result = await service.getInProgressLiveSessions(studentId);

    expect(liveSessionModel.find).toHaveBeenCalledWith({
      student: new Types.ObjectId(studentId),
      status: SessionStatus.Active,
    });
    expect(query.populate).toHaveBeenCalledWith('topic');
    expect(query.sort).toHaveBeenCalledWith({ updatedAt: -1 });
    expect(result[0].id).toBe(liveSessionId.toString());
    expect(result[0].topic).toEqual({
      id: topicId.toString(),
      slug: 'memory-management',
      title: 'Memory Management',
      description: 'Memory concepts.',
      tags: ['systems'],
    });
  });

  it('deletes live sessions owned by the authenticated user', async () => {
    liveSessionModel.findOneAndDelete.mockReturnValue(
      execQuery({ _id: liveSessionId }),
    );

    await service.deleteLiveSession(liveSessionId.toString(), studentId);

    expect(liveSessionModel.findOneAndDelete).toHaveBeenCalledWith({
      _id: liveSessionId,
      student: new Types.ObjectId(studentId),
    });
  });

  it('rejects deleting missing live sessions', async () => {
    liveSessionModel.findOneAndDelete.mockReturnValue(execQuery(null));

    await expect(
      service.deleteLiveSession(liveSessionId.toString(), studentId),
    ).rejects.toThrow(NotFoundException);
  });

  it('continues a live session by generating the next required question', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(sortableQuery(null));
    liveQuestionModel.find.mockReturnValue(
      sortableQuery([
        { question: questionOne, status: LiveQuestionStatus.Passed },
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

  it('continues a failed live set by returning the failed question', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(sortableQuery(null));
    liveQuestionModel.find.mockReturnValue(
      sortableQuery([
        { question: questionOne, status: LiveQuestionStatus.Passed },
        { question: questionTwo, status: LiveQuestionStatus.Passed },
        {
          _id: liveQuestionId,
          question: questionThree,
          status: LiveQuestionStatus.Failed,
        },
      ]),
    );
    const result = await service.continueLiveSession(
      studentId,
      liveSessionId.toString(),
    );

    expect(result).toEqual({
      sessionId: liveSessionId.toString(),
      questionId: liveQuestionId.toString(),
      question: questionThree,
    });
    expect(liveQuestionModel.updateOne).not.toHaveBeenCalled();
    expect(generateLiveQuestion).not.toHaveBeenCalled();
    expect(liveQuestionModel.create).not.toHaveBeenCalled();
  });

  it('continues an early failed live question without generating', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(sortableQuery(null));
    liveQuestionModel.find.mockReturnValue(
      sortableQuery([
        {
          _id: liveQuestionId,
          question: questionOne,
          status: LiveQuestionStatus.Failed,
        },
      ]),
    );

    const result = await service.continueLiveSession(
      studentId,
      liveSessionId.toString(),
    );

    expect(result).toEqual({
      sessionId: liveSessionId.toString(),
      questionId: liveQuestionId.toString(),
      question: questionOne,
    });
    expect(generateLiveQuestion).not.toHaveBeenCalled();
    expect(liveQuestionModel.create).not.toHaveBeenCalled();
  });

  it('rejects pending live questions and returns a replacement', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(
      execQuery({
        _id: liveQuestionId,
        question: questionOne,
        questionNumber: 1,
        status: LiveQuestionStatus.Pending,
      }),
    );
    liveQuestionModel.updateOne.mockReturnValue(
      execQuery({ modifiedCount: 1 }),
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
    expect(liveQuestionModel.updateOne).toHaveBeenCalledWith(
      expect.objectContaining({
        _id: liveQuestionId,
        status: LiveQuestionStatus.Pending,
      }),
      { $set: { status: LiveQuestionStatus.Rejected } },
    );
    expect(generateLiveQuestion.mock.invocationCallOrder[0]).toBeLessThan(
      liveQuestionModel.updateOne.mock.invocationCallOrder[0],
    );
  });

  it('rejects failed live questions and returns a replacement', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(
      execQuery({
        _id: liveQuestionId,
        question: questionThree,
        questionNumber: 3,
        status: LiveQuestionStatus.Failed,
      }),
    );
    liveQuestionModel.updateOne.mockReturnValue(
      execQuery({ modifiedCount: 1 }),
    );
    topicModel.findById.mockReturnValue(execQuery(topic));
    liveQuestionModel.find.mockReturnValue(
      sortableQuery([
        { question: questionOne, status: LiveQuestionStatus.Passed },
        { question: questionTwo, status: LiveQuestionStatus.Passed },
      ]),
    );
    liveQuestionModel.create.mockResolvedValue({ _id: replacementQuestionId });
    generateLiveQuestion.mockResolvedValue(questionThree);

    await service.rejectLiveQuestion(
      studentId,
      liveSessionId.toString(),
      liveQuestionId.toString(),
      'Still confusing.',
    );

    expect(liveQuestionModel.updateOne).toHaveBeenCalledWith(
      expect.objectContaining({
        _id: liveQuestionId,
        status: LiveQuestionStatus.Failed,
      }),
      { $set: { status: LiveQuestionStatus.Rejected } },
    );
    expect(liveQuestionModel.create).toHaveBeenCalledWith(
      expect.objectContaining({ questionNumber: 3 }),
    );
  });

  it('keeps the pending question when replacement generation fails', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(
      execQuery({
        _id: liveQuestionId,
        question: questionOne,
        questionNumber: 1,
        status: LiveQuestionStatus.Pending,
      }),
    );
    topicModel.findById.mockReturnValue(execQuery(topic));
    liveQuestionModel.find.mockReturnValue(sortableQuery([]));
    generateLiveQuestion.mockRejectedValue(new Error('AI failed'));

    await expect(
      service.rejectLiveQuestion(
        studentId,
        liveSessionId.toString(),
        liveQuestionId.toString(),
        'Ambiguous.',
      ),
    ).rejects.toThrow('AI failed');

    expect(liveQuestionModel.updateOne).not.toHaveBeenCalled();
    expect(liveQuestionModel.create).not.toHaveBeenCalled();
  });

  it('rejects stale live question rejection requests', async () => {
    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, currentLevel: 0 }),
    );
    liveQuestionModel.findOne.mockReturnValue(execQuery(null));

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
      expect.objectContaining({
        status: {
          $in: [LiveQuestionStatus.Pending, LiveQuestionStatus.Failed],
        },
      }),
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
    expect(result.answers[0].answerText).toBe('Pages');
    expect(result.answers[0].correctAnswerText).toBe('Pages');
    expect(result.nextQuestion?.question).toEqual(questionTwo);
  });

  it('returns a failed live answer without generating another question', async () => {
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
        { question: questionOne, answer: createAnswer(questionOne, 'o2') },
      ]),
    );

    const result = await service.submitLiveAnswer(
      studentId,
      liveSessionId.toString(),
      liveQuestionId.toString(),
      'o2',
    );

    expect(result.answers).toHaveLength(1);
    expect(result.answers[0].score).toBe(0);
    expect(result.answers[0].answerText).toBe('Threads');
    expect(result.answers[0].correctAnswerText).toBe('Pages');
    expect(result.nextQuestion).toBeNull();
    expect(topicModel.findById).not.toHaveBeenCalled();
    expect(generateLiveQuestion).not.toHaveBeenCalled();
    expect(liveQuestionModel.create).not.toHaveBeenCalled();
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
    setAttemptModel.create.mockImplementation(
      (payload: { liveSession?: Types.ObjectId; session?: Types.ObjectId }) => {
        expect(payload.liveSession).toEqual(liveSessionId);
        expect(payload).not.toHaveProperty('session');

        return Promise.resolve(
          createAttempt([questionOne, questionTwo, questionThree], true),
        );
      },
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

  it('passes recent accepted context when generating next live level', async () => {
    const acceptedLiveQuestions = [
      { question: questionOne, answer: createAnswer(questionOne, 'o1') },
      { question: questionTwo, answer: createAnswer(questionTwo, 'o1') },
      { question: questionThree, answer: createAnswer(questionThree, 'o1') },
    ];
    const recentLiveQuestions = [
      { level: 7, question: questionOne },
      { level: 8, question: questionTwo },
      { level: 9, question: questionThree },
    ];

    liveSessionModel.findOne.mockReturnValue(
      execQuery({ _id: liveSessionId, topic: topicId, currentLevel: 9 }),
    );
    liveQuestionModel.findOne.mockReturnValue(
      execQuery({ _id: liveQuestionId, question: questionThree }),
    );
    liveQuestionModel.updateOne.mockReturnValue(
      execQuery({ modifiedCount: 1 }),
    );
    liveQuestionModel.find
      .mockReturnValueOnce(sortableQuery(acceptedLiveQuestions))
      .mockReturnValueOnce(sortableQuery(recentLiveQuestions));
    questionSetModel.create.mockResolvedValue(createQuestionSet(9));
    setAttemptModel.create.mockResolvedValue(
      createAttempt([questionOne, questionTwo, questionThree], true),
    );
    topicModel.findById.mockReturnValue(execQuery(topic));
    liveQuestionModel.create.mockResolvedValue({ _id: replacementQuestionId });
    liveSessionModel.updateOne.mockReturnValue(execQuery({ modifiedCount: 1 }));

    await service.submitLiveAnswer(
      studentId,
      liveSessionId.toString(),
      liveQuestionId.toString(),
      'o1',
    );

    expect(generateLiveQuestion).toHaveBeenCalledWith(
      expect.objectContaining({
        level: 10,
        recentAcceptedQuestions: [
          {
            level: 7,
            prompt: questionOne.prompt,
            targetConcepts: questionOne.targetConcepts,
          },
          {
            level: 8,
            prompt: questionTwo.prompt,
            targetConcepts: questionTwo.targetConcepts,
          },
          {
            level: 9,
            prompt: questionThree.prompt,
            targetConcepts: questionThree.targetConcepts,
          },
        ],
      }),
    );
  });

  it('creates live session evaluations with a live session reference', async () => {
    const attempt = createAttempt(
      [questionOne, questionTwo, questionThree],
      true,
    );
    const evaluation = {
      strengths: ['paging'],
      weaknesses: [],
      recommendations: [],
    };

    liveSessionModel.findOne.mockReturnValue(
      execQuery({
        _id: liveSessionId,
        student: new Types.ObjectId(studentId),
        topic: topicId,
        currentLevel: 10,
      }),
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
    questionSetModel.create.mockResolvedValue(createQuestionSet(10));
    setAttemptModel.create.mockResolvedValue(attempt);
    setAttemptModel.find.mockReturnValue(execQuery([attempt]));
    sessionEvaluationModel.create.mockImplementation(
      (payload: { liveSession?: Types.ObjectId; session?: Types.ObjectId }) => {
        expect(payload.liveSession).toEqual(liveSessionId);
        expect(payload).not.toHaveProperty('session');

        return Promise.resolve(evaluation);
      },
    );
    sessionEvaluationModel.find.mockReturnValue(execQuery([evaluation]));
    topicModel.findById.mockReturnValue(execQuery(topic));
    liveQuestionModel.create.mockResolvedValue({ _id: replacementQuestionId });
    generateLiveQuestion.mockResolvedValue(nextLevelQuestion);
    liveSessionModel.updateOne.mockReturnValue(execQuery({ modifiedCount: 1 }));

    await service.submitLiveAnswer(
      studentId,
      liveSessionId.toString(),
      liveQuestionId.toString(),
      'o1',
    );

    expect(setAttemptModel.find).toHaveBeenCalledWith({
      liveSession: liveSessionId,
      level: { $gte: 0, $lte: 10 },
    });
    expect(sessionEvaluationModel.find).toHaveBeenCalledWith({
      liveSession: liveSessionId,
    });
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
    setAttemptModel.create.mockImplementation(
      (payload: { liveSession?: Types.ObjectId; session?: Types.ObjectId }) => {
        expect(payload.session).toEqual(sessionId);
        expect(payload).not.toHaveProperty('liveSession');

        return Promise.resolve(attempt);
      },
    );
    sessionModel.updateOne.mockReturnValue(execQuery({ modifiedCount: 1 }));
    questionSetModel.findOne.mockReturnValue(execQuery(nextQuestionSet));

    const result = await service.submitAnswer(
      studentId,
      sessionId.toString(),
      questionSetId.toString(),
      [{ questionId: questionOne.id, answer: 'o1' }],
    );

    expect(setAttemptModel.create).toHaveBeenCalled();
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

  const populateSortableQuery = (
    value: unknown,
  ): { exec: jest.Mock; populate: jest.Mock; sort: jest.Mock } => {
    const query = {
      exec: jest.fn().mockResolvedValue(value),
      populate: jest.fn(),
      sort: jest.fn(),
    };

    query.populate.mockReturnValue(query);
    query.sort.mockReturnValue(query);

    return query;
  };

  const createAnswer = (question: Question, answer: string): unknown => ({
    id: `ans-${question.id}`,
    questionId: question.id,
    questionPrompt: question.prompt,
    questionType: question.type,
    answer,
    answerText:
      question.options?.find((option) => option.id === answer)?.text ?? answer,
    correctAnswer: question.correctOptionId,
    correctAnswerText:
      question.options?.find((option) => option.id === question.correctOptionId)
        ?.text ?? question.correctOptionId,
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
