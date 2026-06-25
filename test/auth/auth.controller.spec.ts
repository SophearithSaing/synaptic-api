import { ConfigService } from '@nestjs/config';
import type { Request, Response } from 'express';
import {
  ACCESS_TOKEN_COOKIE_NAME,
  REFRESH_TOKEN_COOKIE_NAME,
} from '../../src/auth/auth-cookie';
import { AuthController } from '../../src/auth/auth.controller';
import {
  AuthSessionResponse,
  AuthService,
} from '../../src/auth/auth.service';
import { UserRole } from '../../src/auth/schemas/user.schema';
import { RequestWithUser } from '../../src/auth/types/request-with-user.type';

describe('AuthController', () => {
  let controller: AuthController;
  let service: jest.Mocked<
    Pick<AuthService, 'register' | 'login' | 'refresh' | 'logout'>
  >;
  let configService: jest.Mocked<Pick<ConfigService, 'getOrThrow'>>;
  let response: jest.Mocked<Pick<Response, 'cookie' | 'clearCookie'>>;
  const authResponse: AuthSessionResponse = {
    access_token: 'signed-token',
    refresh_token: 'refresh-token',
  };

  beforeEach(() => {
    service = {
      register: jest.fn(),
      login: jest.fn(),
      refresh: jest.fn(),
      logout: jest.fn(),
    };
    configService = {
      getOrThrow: jest.fn().mockReturnValue('15m'),
    };
    response = {
      cookie: jest.fn(),
      clearCookie: jest.fn(),
    };
    controller = new AuthController(
      service as unknown as AuthService,
      configService as unknown as ConfigService,
    );
  });

  it('registers users and sets auth cookies', async () => {
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
    ).resolves.toEqual({ access_token: 'signed-token' });

    expect(service.register).toHaveBeenCalledWith(
      'student@example.com',
      'student-user',
      'password123',
    );
    expect(response.cookie).toHaveBeenCalledWith(
      ACCESS_TOKEN_COOKIE_NAME,
      'signed-token',
      expect.any(Object),
    );
    expect(response.cookie).toHaveBeenCalledWith(
      REFRESH_TOKEN_COOKIE_NAME,
      'refresh-token',
      expect.any(Object),
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

  it('refreshes sessions and sets rotated auth cookies', async () => {
    service.refresh.mockResolvedValue(authResponse);
    const request = {
      cookies: { [REFRESH_TOKEN_COOKIE_NAME]: 'old-refresh-token' },
    } as unknown as Request;

    await expect(
      controller.refresh(request as never, response as unknown as Response),
    ).resolves.toEqual({ access_token: 'signed-token' });

    expect(service.refresh).toHaveBeenCalledWith('old-refresh-token');
    expect(response.cookie).toHaveBeenCalledWith(
      ACCESS_TOKEN_COOKIE_NAME,
      'signed-token',
      expect.any(Object),
    );
    expect(response.cookie).toHaveBeenCalledWith(
      REFRESH_TOKEN_COOKIE_NAME,
      'refresh-token',
      expect.any(Object),
    );
  });

  it('logs out sessions and clears auth cookies', async () => {
    const request = {
      cookies: { [REFRESH_TOKEN_COOKIE_NAME]: 'refresh-token' },
    } as unknown as Request;

    await controller.logout(request as never, response as unknown as Response);

    expect(service.logout).toHaveBeenCalledWith('refresh-token');
    expect(response.clearCookie).toHaveBeenCalledWith(
      ACCESS_TOKEN_COOKIE_NAME,
      expect.any(Object),
    );
    expect(response.clearCookie).toHaveBeenCalledWith(
      REFRESH_TOKEN_COOKIE_NAME,
      expect.any(Object),
    );
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
    ).resolves.toEqual({ access_token: 'signed-token' });

    expect(service.login).toHaveBeenCalledWith(
      'student@example.com',
      'password123',
    );
    expect(response.cookie).toHaveBeenCalledWith(
      ACCESS_TOKEN_COOKIE_NAME,
      'signed-token',
      expect.any(Object),
    );
    expect(response.cookie).toHaveBeenCalledWith(
      REFRESH_TOKEN_COOKIE_NAME,
      'refresh-token',
      expect.any(Object),
    );
  });
});
