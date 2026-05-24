import { SessionsController } from './sessions.controller';
import { SessionsService } from './sessions.service';
import { AiProvider } from '../ai/types/ai.types';
import { RequestWithUser } from '../auth/types/request-with-user.type';
import { UserRole } from '../auth/schemas/user.schema';

describe('SessionsController', () => {
  let controller: SessionsController;
  let service: jest.Mocked<
    Pick<SessionsService, 'startSession' | 'submitSession'>
  >;
  const request = {
    user: {
      email: 'student@example.com',
      role: UserRole.User,
      userId: 'authenticated-user-id',
    },
  } as RequestWithUser;

  beforeEach(() => {
    service = {
      startSession: jest.fn(),
      submitSession: jest.fn(),
    };
    controller = new SessionsController(service as unknown as SessionsService);
  });

  it('starts sessions with the authenticated user id', async () => {
    service.startSession.mockResolvedValue('session' as never);

    await controller.startSession(request, {
      provider: AiProvider.Gemini,
      topicId: '507f1f77bcf86cd799439011',
    });

    expect(service.startSession).toHaveBeenCalledWith(
      '507f1f77bcf86cd799439011',
      'authenticated-user-id',
      AiProvider.Gemini,
    );
  });

  it('submits sessions with the authenticated user id', async () => {
    const answers = [
      {
        questionId: '507f1f77bcf86cd799439012',
        studentAnswer: 'answer',
      },
    ];
    service.submitSession.mockResolvedValue('result' as never);

    await controller.submitSession(request, '507f1f77bcf86cd799439013', {
      answers,
      provider: AiProvider.Claude,
    });

    expect(service.submitSession).toHaveBeenCalledWith(
      '507f1f77bcf86cd799439013',
      'authenticated-user-id',
      answers,
      AiProvider.Claude,
    );
  });
});
