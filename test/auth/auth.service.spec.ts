import { ConflictException, UnauthorizedException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { JwtService } from '@nestjs/jwt';
import * as bcrypt from 'bcrypt';
import { Types } from 'mongoose';
import { AuthService } from '../../src/auth/auth.service';
import { UserRole } from '../../src/auth/schemas/user.schema';

jest.mock('bcrypt', () => ({
  compare: jest.fn(),
  genSalt: jest.fn(),
  hash: jest.fn(),
}));

interface MockUserDocument {
  _id: { toString: () => string };
  email: string;
  username: string;
  password: string;
  role?: UserRole;
}

describe('AuthService', () => {
  let service: AuthService;
  let jwtService: jest.Mocked<Pick<JwtService, 'sign'>>;
  let configService: jest.Mocked<Pick<ConfigService, 'getOrThrow'>>;
  let save: jest.Mock;
  let sessionSave: jest.Mock;
  let findOne: jest.Mock;
  let findById: jest.Mock;
  let findSessionById: jest.Mock;
  let findByIdAndUpdate: jest.Mock;
  let userModel: jest.Mock & { findOne: jest.Mock; findById: jest.Mock };
  let authSessionModel: jest.Mock & {
    findById: jest.Mock;
    findByIdAndUpdate: jest.Mock;
  };

  beforeEach(() => {
    save = jest.fn();
    sessionSave = jest.fn().mockResolvedValue({ id: 'session-id' });
    findOne = jest.fn();
    findById = jest.fn();
    findSessionById = jest.fn();
    findByIdAndUpdate = jest.fn();
    userModel = Object.assign(
      jest.fn().mockImplementation((data: Record<string, unknown>) => ({
        ...data,
        save,
      })),
      { findById, findOne },
    );
    authSessionModel = Object.assign(
      jest.fn().mockImplementation((data: Record<string, unknown>) => ({
        ...data,
        save: sessionSave,
      })),
      { findById: findSessionById, findByIdAndUpdate },
    );
    jwtService = {
      sign: jest.fn().mockReturnValue('signed-token'),
    };
    configService = {
      getOrThrow: jest.fn().mockReturnValue('7d'),
    };
    service = new AuthService(
      userModel as never,
      authSessionModel as never,
      jwtService as unknown as JwtService,
      configService as unknown as ConfigService,
    );

    jest.mocked(bcrypt.genSalt).mockResolvedValue('salt' as never);
    jest
      .mocked(bcrypt.hash)
      .mockImplementation(async (value: string) => `hashed-${value}` as never);
    jest.mocked(bcrypt.compare).mockResolvedValue(true as never);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('hashes passwords, normalizes input, and creates sessions', async () => {
    const savedUser: MockUserDocument = {
      _id: { toString: () => '507f1f77bcf86cd799439011' },
      email: 'student@example.com',
      username: 'student-user',
      password: 'hashed-Password1',
      role: UserRole.User,
    };
    save.mockResolvedValue(savedUser);

    const result = await service.register(
      ' Student@Example.COM ',
      ' student-user ',
      'Password1',
    );

    expect(result.access_token).toBe('signed-token');
    expect(result.refresh_token).toMatch(/^session-id\./);
    expect(userModel).toHaveBeenCalledWith({
      email: 'student@example.com',
      username: 'student-user',
      password: 'hashed-Password1',
    });
    expect(authSessionModel).toHaveBeenCalledWith({
      expiresAt: expect.any(Date),
      refreshTokenHash: expect.stringMatching(/^hashed-/),
      userId: expect.any(Types.ObjectId),
    });
  });

  it('throws conflict exceptions for duplicate usernames', async () => {
    save.mockRejectedValue({ code: 11000, keyPattern: { username: 1 } });

    await expect(
      service.register('student@example.com', 'student', 'Password1'),
    ).rejects.toThrow(new ConflictException('Username already exists'));
  });

  it('throws conflict exceptions for duplicate emails', async () => {
    save.mockRejectedValue({ code: 11000, keyPattern: { email: 1 } });

    await expect(
      service.register('student@example.com', 'student', 'Password1'),
    ).rejects.toThrow(new ConflictException('Email already exists'));
  });

  it('normalizes email identifiers and selects passwords during login', async () => {
    const user: MockUserDocument = {
      _id: { toString: () => '507f1f77bcf86cd799439011' },
      email: 'student@example.com',
      username: 'student-user',
      password: 'hashed-password',
      role: UserRole.User,
    };
    const select = jest.fn().mockResolvedValue(user);
    findOne.mockReturnValue({ select });

    const result = await service.login(' Student@Example.COM ', 'Password1');

    expect(result.access_token).toBe('signed-token');
    expect(result.refresh_token).toMatch(/^session-id\./);
    expect(findOne).toHaveBeenCalledWith({ email: 'student@example.com' });
    expect(select).toHaveBeenCalledWith('+password');
    expect(bcrypt.compare).toHaveBeenCalledWith('Password1', 'hashed-password');
  });

  it('uses username identifiers with case-insensitive collation', async () => {
    const user: MockUserDocument = {
      _id: { toString: () => '507f1f77bcf86cd799439011' },
      email: 'student@example.com',
      username: 'student-user',
      password: 'hashed-password',
      role: UserRole.User,
    };
    const select = jest.fn().mockResolvedValue(user);
    const collation = jest.fn();
    findOne.mockReturnValue({ collation, select });

    await expect(service.login(' Student-User ', 'Password1')).resolves.toEqual(
      expect.objectContaining({ access_token: 'signed-token' }),
    );

    expect(findOne).toHaveBeenCalledWith({ username: 'Student-User' });
    expect(collation).toHaveBeenCalledWith({ locale: 'en', strength: 2 });
    expect(select).toHaveBeenCalledWith('+password');
  });

  it('refreshes sessions and rotates refresh tokens', async () => {
    const session = {
      id: 'session-id',
      userId: 'user-id',
      refreshTokenHash: 'hashed-old-secret',
      expiresAt: new Date(Date.now() + 60000),
      save: jest.fn(),
    };
    const user: MockUserDocument = {
      _id: { toString: () => 'user-id' },
      email: 'student@example.com',
      username: 'student-user',
      password: 'hashed-password',
    };
    findSessionById.mockReturnValue({
      select: jest.fn().mockResolvedValue(session),
    });
    findById.mockResolvedValue(user);

    const result = await service.refresh('session-id.old-secret');

    expect(result.access_token).toBe('signed-token');
    expect(result.refresh_token).toMatch(/^session-id\./);
    expect(bcrypt.compare).toHaveBeenCalledWith(
      'old-secret',
      'hashed-old-secret',
    );
    expect(session.save).toHaveBeenCalled();
  });

  it('rejects invalid refresh sessions', async () => {
    findSessionById.mockReturnValue({
      select: jest.fn().mockResolvedValue(null),
    });

    await expect(service.refresh('session-id.secret')).rejects.toThrow(
      UnauthorizedException,
    );
  });

  it('revokes refresh sessions during logout', async () => {
    await service.logout('session-id.secret');

    expect(findByIdAndUpdate).toHaveBeenCalledWith('session-id', {
      revokedAt: expect.any(Date),
    });
  });

  it('rejects login when no user exists', async () => {
    const select = jest.fn().mockResolvedValue(null);
    findOne.mockReturnValue({ select });

    await expect(
      service.login('student@example.com', 'Password1'),
    ).rejects.toThrow(UnauthorizedException);
  });

  it('rejects login when the password does not match', async () => {
    const user: MockUserDocument = {
      _id: { toString: () => '507f1f77bcf86cd799439011' },
      email: 'student@example.com',
      username: 'student-user',
      password: 'hashed-password',
    };
    const select = jest.fn().mockResolvedValue(user);
    findOne.mockReturnValue({ select });
    jest.mocked(bcrypt.compare).mockResolvedValue(false as never);

    await expect(
      service.login('student@example.com', 'WrongPassword1'),
    ).rejects.toThrow(UnauthorizedException);
  });
});
