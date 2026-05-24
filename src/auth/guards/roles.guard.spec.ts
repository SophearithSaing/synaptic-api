import { ExecutionContext } from '@nestjs/common';
import { Reflector } from '@nestjs/core';
import { RolesGuard } from './roles.guard';
import { UserRole } from '../schemas/user.schema';
import { RequestWithUser } from '../types/request-with-user.type';

describe('RolesGuard', () => {
  const getContext = (role?: UserRole): ExecutionContext =>
    ({
      getClass: jest.fn(),
      getHandler: jest.fn(),
      switchToHttp: () => ({
        getRequest: (): Partial<RequestWithUser> => ({
          user: role
            ? {
                email: 'test@example.com',
                role,
                userId: 'user-id',
              }
            : undefined,
        }),
      }),
    }) as unknown as ExecutionContext;

  it('allows routes without role metadata', () => {
    const reflector = {
      getAllAndOverride: jest.fn().mockReturnValue(undefined),
    } as unknown as Reflector;
    const guard = new RolesGuard(reflector);

    expect(guard.canActivate(getContext(UserRole.User))).toBe(true);
  });

  it('allows matching roles', () => {
    const reflector = {
      getAllAndOverride: jest.fn().mockReturnValue([UserRole.Admin]),
    } as unknown as Reflector;
    const guard = new RolesGuard(reflector);

    expect(guard.canActivate(getContext(UserRole.Admin))).toBe(true);
  });

  it('rejects non-matching roles', () => {
    const reflector = {
      getAllAndOverride: jest.fn().mockReturnValue([UserRole.Admin]),
    } as unknown as Reflector;
    const guard = new RolesGuard(reflector);

    expect(guard.canActivate(getContext(UserRole.User))).toBe(false);
  });

  it('rejects missing authenticated users', () => {
    const reflector = {
      getAllAndOverride: jest.fn().mockReturnValue([UserRole.Admin]),
    } as unknown as Reflector;
    const guard = new RolesGuard(reflector);

    expect(guard.canActivate(getContext())).toBe(false);
  });
});
