import { Types } from 'mongoose';
import { AiService } from '../../src/ai/ai.service';
import { AiLogOperation } from '../../src/ai/schemas/ai-log.schema';
import { LiveQuestionStatus } from '../../src/sessions/schemas/live-question.schema';

describe('AiService', () => {
  const aiLogId = new Types.ObjectId('507f1f77bcf86cd799439011');
  const liveQuestionId = new Types.ObjectId('507f1f77bcf86cd799439012');
  let service: AiService;
  let aiLogModel: Record<string, jest.Mock>;

  beforeEach(() => {
    aiLogModel = {
      create: jest.fn(),
      updateOne: jest.fn(),
      find: jest.fn(),
      countDocuments: jest.fn(),
    };
    service = new AiService(aiLogModel as never);
  });

  it('records a raw question-generation completion', async () => {
    aiLogModel.create.mockResolvedValue({ _id: aiLogId });

    const result = await service.createAiLog(
      AiLogOperation.QuestionGeneration,
      'openai/gpt-oss-120b',
      '{"topic":"memory-management"}',
      '{"question":{}}',
    );

    expect(result).toEqual(aiLogId);
    expect(aiLogModel.create).toHaveBeenCalledWith({
      operation: AiLogOperation.QuestionGeneration,
      aiModel: 'openai/gpt-oss-120b',
      prompt: '{"topic":"memory-management"}',
      output: '{"question":{}}',
    });
  });

  it('links a log to its persisted live question', async () => {
    const query = execQuery({ modifiedCount: 1 });
    aiLogModel.updateOne.mockReturnValue(query);

    await service.linkLiveQuestion(aiLogId, liveQuestionId);

    expect(aiLogModel.updateOne).toHaveBeenCalledWith(
      { _id: aiLogId },
      { $set: { liveQuestion: liveQuestionId } },
    );
    expect(query.exec).toHaveBeenCalled();
  });

  it('returns newest logs with populated live question metadata', async () => {
    const createdAt = new Date('2026-07-29T00:00:00.000Z');
    const query = paginatedQuery([
      {
        _id: aiLogId,
        operation: AiLogOperation.QuestionGeneration,
        aiModel: 'openai/gpt-oss-120b',
        prompt: 'prompt',
        output: 'output',
        createdAt,
        liveQuestion: {
          _id: liveQuestionId,
          level: 2,
          questionNumber: 3,
          status: LiveQuestionStatus.Rejected,
          question: { id: 'q1', prompt: 'What is paging?' },
        },
      },
    ]);
    aiLogModel.find.mockReturnValue(query);
    aiLogModel.countDocuments.mockReturnValue(execQuery(41));

    const result = await service.getAiLogs(3, 20);

    expect(query.sort).toHaveBeenCalledWith({ createdAt: -1 });
    expect(query.skip).toHaveBeenCalledWith(40);
    expect(query.limit).toHaveBeenCalledWith(20);
    expect(query.populate).toHaveBeenCalledWith('liveQuestion');
    expect(result).toEqual({
      items: [
        {
          id: aiLogId.toString(),
          operation: AiLogOperation.QuestionGeneration,
          aiModel: 'openai/gpt-oss-120b',
          prompt: 'prompt',
          output: 'output',
          createdAt,
          liveQuestion: {
            id: liveQuestionId.toString(),
            level: 2,
            questionNumber: 3,
            status: LiveQuestionStatus.Rejected,
            question: { id: 'q1', prompt: 'What is paging?' },
          },
        },
      ],
      total: 41,
      page: 3,
      limit: 20,
    });
  });

  const execQuery = (value: unknown): { exec: jest.Mock } => ({
    exec: jest.fn().mockResolvedValue(value),
  });

  const paginatedQuery = (
    value: unknown,
  ): {
    exec: jest.Mock;
    sort: jest.Mock;
    skip: jest.Mock;
    limit: jest.Mock;
    populate: jest.Mock;
  } => {
    const query = {
      exec: jest.fn().mockResolvedValue(value),
      sort: jest.fn(),
      skip: jest.fn(),
      limit: jest.fn(),
      populate: jest.fn(),
    };

    query.sort.mockReturnValue(query);
    query.skip.mockReturnValue(query);
    query.limit.mockReturnValue(query);
    query.populate.mockReturnValue(query);

    return query;
  };
});
