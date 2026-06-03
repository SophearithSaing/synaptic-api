import { AuthController } from './auth.controller';
import { AuthResponse, AuthService } from './auth.service';

describe('AuthController', () => {
  let controller: AuthController;
  let service: jest.Mocked<Pick<AuthService, 'register' | 'login'>>;
  const authResponse: AuthResponse = {
    access_token: 'signed-token',
  };

  beforeEach(() => {
    service = {
      register: jest.fn(),
      login: jest.fn(),
    };
    controller = new AuthController(service as unknown as AuthService);
  });

  it('registers users with username and returns an access token', async () => {
    service.register.mockResolvedValue(authResponse);

    await expect(
      controller.register({
        email: 'student@example.com',
        password: 'password123',
        username: 'student-user',
      }),
    ).resolves.toEqual(authResponse);

    expect(service.register).toHaveBeenCalledWith(
      'student@example.com',
      'student-user',
      'password123',
    );
  });

  it('logs users in with email and password', async () => {
    service.login.mockResolvedValue(authResponse);

    await expect(
      controller.login({
        email: 'student@example.com',
        password: 'password123',
      }),
    ).resolves.toEqual(authResponse);

    expect(service.login).toHaveBeenCalledWith(
      'student@example.com',
      'password123',
    );
  });
});
