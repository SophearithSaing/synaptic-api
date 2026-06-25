import { ExecutionContext, ForbiddenException } from '@nestjs/common';
import {
  CSRF_TOKEN_COOKIE_NAME,
  CSRF_TOKEN_HEADER_NAME,
} from '../../../src/auth/auth-cookie';
import { CsrfGuard } from '../../../src/auth/guards/csrf.guard';

describe('CsrfGuard', () => {
  const getContext = (
    method: string,
    cookieToken?: string,
    headerToken?: string,
  ): ExecutionContext =>
    ({
      switchToHttp: () => ({
        getRequest: () => ({
          cookies: { [CSRF_TOKEN_COOKIE_NAME]: cookieToken },
          headers: { [CSRF_TOKEN_HEADER_NAME]: headerToken },
          method,
        }),
      }),
    }) as unknown as ExecutionContext;

  it('allows safe requests without CSRF tokens', () => {
    const guard = new CsrfGuard();

    expect(guard.canActivate(getContext('GET'))).toBe(true);
  });

  it('allows mutating requests with matching CSRF tokens', () => {
    const guard = new CsrfGuard();

    expect(guard.canActivate(getContext('POST', 'token', 'token'))).toBe(true);
  });

  it('rejects mutating requests with missing CSRF tokens', () => {
    const guard = new CsrfGuard();

    expect(() => guard.canActivate(getContext('POST'))).toThrow(
      ForbiddenException,
    );
  });

  it('rejects mutating requests with mismatched CSRF tokens', () => {
    const guard = new CsrfGuard();

    expect(() => guard.canActivate(getContext('PATCH', 'one', 'two'))).toThrow(
      ForbiddenException,
    );
  });
});
