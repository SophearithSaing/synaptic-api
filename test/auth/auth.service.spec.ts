import { ConflictException, UnauthorizedException } from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import * as bcrypt from 'bcrypt';
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
  let save: jest.Mock;
  let findOne: jest.Mock;
  let userModel: jest.Mock & { findOne: jest.Mock };

  beforeEach(() => {
    save = jest.fn();
    findOne = jest.fn();
    userModel = Object.assign(
      jest.fn().mockImplementation((data: Record<string, unknown>) => ({
        ...data,
        save,
      })),
      { findOne },
    );
    jwtService = {
      sign: jest.fn().mockReturnValue('signed-token'),
    };
    service = new AuthService(
      userModel as never,
      jwtService as unknown as JwtService,
    );

    jest.mocked(bcrypt.genSalt).mockResolvedValue('salt' as never);
    jest.mocked(bcrypt.hash).mockResolvedValue('hashed-password' as never);
    jest.mocked(bcrypt.compare).mockResolvedValue(true as never);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('hashes passwords and normalizes user input during registration', async () => {
    const savedUser: MockUserDocument = {
      _id: { toString: () => 'user-id' },
      email: 'student@example.com',
      username: 'student-user',
      password: 'hashed-password',
      role: UserRole.User,
    };
    save.mockResolvedValue(savedUser);

    await expect(
      service.register(' Student@Example.COM ', ' student-user ', 'Password1'),
    ).resolves.toEqual({ access_token: 'signed-token' });

    expect(bcrypt.genSalt).toHaveBeenCalled();
    expect(bcrypt.hash).toHaveBeenCalledWith('Password1', 'salt');
    expect(userModel).toHaveBeenCalledWith({
      email: 'student@example.com',
      username: 'student-user',
      password: 'hashed-password',
    });
    expect(jwtService.sign).toHaveBeenCalledWith({
      email: 'student@example.com',
      username: 'student-user',
      sub: 'user-id',
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
      _id: { toString: () => 'user-id' },
      email: 'student@example.com',
      username: 'student-user',
      password: 'hashed-password',
      role: UserRole.User,
    };
    const select = jest.fn().mockResolvedValue(user);
    findOne.mockReturnValue({ select });

    await expect(
      service.login(' Student@Example.COM ', 'Password1'),
    ).resolves.toEqual({ access_token: 'signed-token' });

    expect(findOne).toHaveBeenCalledWith({ email: 'student@example.com' });
    expect(select).toHaveBeenCalledWith('+password');
    expect(bcrypt.compare).toHaveBeenCalledWith('Password1', 'hashed-password');
    expect(jwtService.sign).toHaveBeenCalledWith({
      email: 'student@example.com',
      username: 'student-user',
      sub: 'user-id',
    });
  });

  it('uses username identifiers with case-insensitive collation', async () => {
    const user: MockUserDocument = {
      _id: { toString: () => 'user-id' },
      email: 'student@example.com',
      username: 'student-user',
      password: 'hashed-password',
      role: UserRole.User,
    };
    const select = jest.fn().mockResolvedValue(user);
    const collation = jest.fn();
    findOne.mockReturnValue({ collation, select });

    await expect(service.login(' Student-User ', 'Password1')).resolves.toEqual(
      { access_token: 'signed-token' },
    );

    expect(findOne).toHaveBeenCalledWith({ username: 'Student-User' });
    expect(collation).toHaveBeenCalledWith({ locale: 'en', strength: 2 });
    expect(select).toHaveBeenCalledWith('+password');
    expect(bcrypt.compare).toHaveBeenCalledWith('Password1', 'hashed-password');
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
      _id: { toString: () => 'user-id' },
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
