import { UnauthorizedException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { JwtStrategy } from '../../../src/auth/strategies/jwt.strategy';
import { UserRole } from '../../../src/auth/schemas/user.schema';

describe('JwtStrategy', () => {
  let configService: jest.Mocked<Pick<ConfigService, 'getOrThrow'>>;
  let findById: jest.Mock;
  let strategy: JwtStrategy;

  beforeEach(() => {
    configService = {
      getOrThrow: jest.fn((key: string) => {
        const values: Record<string, string> = {
          JWT_AUDIENCE: 'synaptic-client',
          JWT_ISSUER: 'synaptic-api',
          JWT_SECRET: 'secret',
        };

        return values[key];
      }),
    };
    findById = jest.fn();
    strategy = new JwtStrategy(
      configService as unknown as ConfigService,
      {
        findById,
      } as never,
    );
  });

  it('requires JWT secret, audience, and issuer configuration', () => {
    expect(configService.getOrThrow).toHaveBeenCalledWith('JWT_AUDIENCE');
    expect(configService.getOrThrow).toHaveBeenCalledWith('JWT_ISSUER');
    expect(configService.getOrThrow).toHaveBeenCalledWith('JWT_SECRET');
  });

  it('returns authenticated users for valid payloads', async () => {
    findById.mockResolvedValue({
      id: 'user-id',
      email: 'student@example.com',
      username: 'student-user',
      role: UserRole.Admin,
    });

    await expect(
      strategy.validate({
        email: 'student@example.com',
        username: 'student-user',
        sub: 'user-id',
      }),
    ).resolves.toEqual({
      email: 'student@example.com',
      username: 'student-user',
      role: UserRole.Admin,
      userId: 'user-id',
    });

    expect(findById).toHaveBeenCalledWith('user-id');
  });

  it('defaults missing roles to user', async () => {
    findById.mockResolvedValue({
      id: 'user-id',
      email: 'student@example.com',
      username: 'student-user',
    });

    await expect(
      strategy.validate({
        email: 'student@example.com',
        username: 'student-user',
        sub: 'user-id',
      }),
    ).resolves.toEqual({
      email: 'student@example.com',
      username: 'student-user',
      role: UserRole.User,
      userId: 'user-id',
    });
  });

  it('rejects missing users', async () => {
    findById.mockResolvedValue(null);

    await expect(
      strategy.validate({
        email: 'deleted@example.com',
        username: 'deleted-user',
        sub: 'deleted-user-id',
      }),
    ).rejects.toThrow(UnauthorizedException);
  });
});
