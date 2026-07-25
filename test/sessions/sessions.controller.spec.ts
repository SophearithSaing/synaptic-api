import { SessionsController } from '../../src/sessions/sessions.controller';
import { SessionsService } from '../../src/sessions/sessions.service';
import { RequestWithUser } from '../../src/auth/types/request-with-user.type';

describe('SessionsController', () => {
  const request = {
    user: {
      userId: '507f1f77bcf86cd799439011',
      email: 'student@example.com',
      username: 'student',
      role: 'user',
    },
  } as RequestWithUser;
  let controller: SessionsController;
  let service: jest.Mocked<
    Pick<
      SessionsService,
      | 'startSession'
      | 'startLiveSession'
      | 'rejectLiveQuestion'
      | 'submitLiveAnswer'
      | 'continueLiveSession'
      | 'getInProgressSessions'
      | 'getInProgressLiveSessions'
      | 'deleteSession'
      | 'continueSession'
      | 'submitAnswer'
    >
  >;

  beforeEach(() => {
    service = {
      startSession: jest.fn(),
      startLiveSession: jest.fn(),
      rejectLiveQuestion: jest.fn(),
      submitLiveAnswer: jest.fn(),
      continueLiveSession: jest.fn(),
      getInProgressSessions: jest.fn(),
      getInProgressLiveSessions: jest.fn(),
      deleteSession: jest.fn(),
      continueSession: jest.fn(),
      submitAnswer: jest.fn(),
    };
    controller = new SessionsController(service as unknown as SessionsService);
  });

  it('starts regular sessions with the authenticated user', async () => {
    service.startSession.mockResolvedValue({} as never);

    await controller.startSession(request, { topicId: 'topic-id' });

    expect(service.startSession).toHaveBeenCalledWith(
      'topic-id',
      request.user.userId,
    );
  });

  it('starts live sessions with the authenticated user', async () => {
    service.startLiveSession.mockResolvedValue({} as never);

    await controller.startLiveSession(request, { topicId: 'topic-id' });

    expect(service.startLiveSession).toHaveBeenCalledWith(
      'topic-id',
      request.user.userId,
    );
  });

  it('rejects live questions with the authenticated user', async () => {
    service.rejectLiveQuestion.mockResolvedValue({} as never);

    await controller.rejectLiveQuestion(request, {
      sessionId: 'session-id',
      questionId: 'question-id',
      reason: 'Ambiguous.',
    });

    expect(service.rejectLiveQuestion).toHaveBeenCalledWith(
      request.user.userId,
      'session-id',
      'question-id',
      'Ambiguous.',
    );
  });

  it('submits live answers with the authenticated user', async () => {
    service.submitLiveAnswer.mockResolvedValue({} as never);

    await controller.submitLiveAnswer(request, {
      sessionId: 'session-id',
      questionId: 'question-id',
      answer: 'answer-id',
    });

    expect(service.submitLiveAnswer).toHaveBeenCalledWith(
      request.user.userId,
      'session-id',
      'question-id',
      'answer-id',
    );
  });

  it('continues live sessions with the authenticated user', async () => {
    service.continueLiveSession.mockResolvedValue({} as never);

    await controller.continueLiveSession(request, { sessionId: 'session-id' });

    expect(service.continueLiveSession).toHaveBeenCalledWith(
      request.user.userId,
      'session-id',
    );
  });

  it('fetches in-progress regular sessions with the authenticated user', async () => {
    service.getInProgressSessions.mockResolvedValue([]);

    await controller.getInProgressSessions(request);

    expect(service.getInProgressSessions).toHaveBeenCalledWith(
      request.user.userId,
    );
  });

  it('fetches in-progress live sessions with the authenticated user', async () => {
    service.getInProgressLiveSessions.mockResolvedValue([]);

    await controller.getInProgressLiveSessions(request);

    expect(service.getInProgressLiveSessions).toHaveBeenCalledWith(
      request.user.userId,
    );
  });

  it('deletes regular sessions with the authenticated user', async () => {
    service.deleteSession.mockResolvedValue(undefined);

    await controller.deleteSession(request, 'session-id');

    expect(service.deleteSession).toHaveBeenCalledWith(
      'session-id',
      request.user.userId,
    );
  });

  it('continues regular sessions with the authenticated user', async () => {
    service.continueSession.mockResolvedValue({} as never);

    await controller.continueSession(request, { sessionId: 'session-id' });

    expect(service.continueSession).toHaveBeenCalledWith(
      'session-id',
      request.user.userId,
    );
  });

  it('submits regular answers with the authenticated user', async () => {
    service.submitAnswer.mockResolvedValue({} as never);
    const answers = [{ questionId: 'q1', answer: 'a1' }];

    await controller.submitAnswer(request, {
      sessionId: 'session-id',
      questionSetId: 'question-set-id',
      answers,
    });

    expect(service.submitAnswer).toHaveBeenCalledWith(
      request.user.userId,
      'session-id',
      'question-set-id',
      answers,
    );
  });
});
