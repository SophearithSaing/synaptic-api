import { ConfigService } from '@nestjs/config';
import type { Response } from 'express';
import { ACCESS_TOKEN_COOKIE_NAME } from '../../src/auth/auth-cookie';
import { AuthController } from '../../src/auth/auth.controller';
import { AuthResponse, AuthService } from '../../src/auth/auth.service';
import { UserRole } from '../../src/auth/schemas/user.schema';
import { RequestWithUser } from '../../src/auth/types/request-with-user.type';

describe('AuthController', () => {
  let controller: AuthController;
  let service: jest.Mocked<Pick<AuthService, 'register' | 'login'>>;
  let configService: jest.Mocked<Pick<ConfigService, 'getOrThrow'>>;
  let response: jest.Mocked<Pick<Response, 'cookie'>>;
  const authResponse: AuthResponse = {
    access_token: 'signed-token',
  };

  beforeEach(() => {
    service = {
      register: jest.fn(),
      login: jest.fn(),
    };
    configService = {
      getOrThrow: jest.fn().mockReturnValue('15m'),
    };
    response = {
      cookie: jest.fn(),
    };
    controller = new AuthController(
      service as unknown as AuthService,
      configService as unknown as ConfigService,
    );
  });

  it('registers users with username and returns an access token', async () => {
    service.register.mockResolvedValue(authResponse);

    await expect(
      controller.register(
        {
          email: 'student@example.com',
          password: 'password123',
          username: 'student-user',
        },
        response as unknown as Response,
      ),
    ).resolves.toEqual(authResponse);

    expect(service.register).toHaveBeenCalledWith(
      'student@example.com',
      'student-user',
      'password123',
    );
    expect(response.cookie).toHaveBeenCalledWith(
      ACCESS_TOKEN_COOKIE_NAME,
      'signed-token',
      {
        httpOnly: true,
        maxAge: 900000,
        path: '/',
        sameSite: 'none',
        secure: true,
      },
    );
  });

  it('returns the current authenticated user', () => {
    const request = {
      user: {
        email: 'student@example.com',
        username: 'student-user',
        userId: 'authenticated-user-id',
        role: UserRole.User,
      },
    } as RequestWithUser;

    expect(controller.getCurrentUser(request)).toEqual(request.user);
  });

  it('logs users in with identifier and password', async () => {
    service.login.mockResolvedValue(authResponse);

    await expect(
      controller.login(
        {
          identifier: 'student@example.com',
          password: 'password123',
        },
        response as unknown as Response,
      ),
    ).resolves.toEqual(authResponse);

    expect(service.login).toHaveBeenCalledWith(
      'student@example.com',
      'password123',
    );
    expect(response.cookie).toHaveBeenCalledWith(
      ACCESS_TOKEN_COOKIE_NAME,
      'signed-token',
      {
        httpOnly: true,
        maxAge: 900000,
        path: '/',
        sameSite: 'none',
        secure: true,
      },
    );
  });
});
