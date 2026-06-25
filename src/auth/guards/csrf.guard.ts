import {
  CanActivate,
  ExecutionContext,
  ForbiddenException,
  Injectable,
} from '@nestjs/common';
import type { Request } from 'express';
import { CSRF_TOKEN_COOKIE_NAME, CSRF_TOKEN_HEADER_NAME } from '../auth-cookie';

interface CsrfRequest extends Request {
  cookies: Record<string, string | undefined>;
}

/**
 * Guard enforcing double-submit CSRF protection for mutating requests.
 */
@Injectable()
export class CsrfGuard implements CanActivate {
  /**
   * Determines whether the request has a valid CSRF token pair.
   *
   * @param context Nest execution context.
   * @returns Whether the request can proceed.
   */
  canActivate(context: ExecutionContext): boolean {
    const request = context.switchToHttp().getRequest<CsrfRequest>();

    if (this.isSafeMethod(request.method)) {
      return true;
    }

    const cookieToken = request.cookies?.[CSRF_TOKEN_COOKIE_NAME];
    const headerToken = this.getHeaderToken(request);

    if (!cookieToken || !headerToken || cookieToken !== headerToken) {
      throw new ForbiddenException('Invalid CSRF token');
    }

    return true;
  }

  /**
   * Extracts the CSRF token header value.
   *
   * @param request Incoming request.
   * @returns Header token, or undefined when missing.
   */
  private getHeaderToken(request: CsrfRequest): string | undefined {
    const value = request.headers[CSRF_TOKEN_HEADER_NAME];

    if (Array.isArray(value)) {
      return value[0];
    }

    return value;
  }

  /**
   * Checks whether an HTTP method is safe from mutation.
   *
   * @param method HTTP method.
   * @returns Whether the method is safe.
   */
  private isSafeMethod(method: string): boolean {
    return ['GET', 'HEAD', 'OPTIONS'].includes(method.toUpperCase());
  }
}
