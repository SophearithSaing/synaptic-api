import { CanActivate, ExecutionContext, Injectable } from '@nestjs/common';
import { Reflector } from '@nestjs/core';
import { ROLES_KEY } from '../decorators/roles.decorator';
import { UserRole } from '../schemas/user.schema';
import { RequestWithUser } from '../types/request-with-user.type';

/**
 * Guard enforcing route-level role requirements.
 */
@Injectable()
export class RolesGuard implements CanActivate {
  /**
   * Creates a roles guard.
   *
   * @param reflector The Nest reflector.
   */
  constructor(private readonly reflector: Reflector) {}

  /**
   * Determines whether the authenticated user has an allowed role.
   *
   * @param context Nest execution context.
   * @returns Whether the request can proceed.
   */
  canActivate(context: ExecutionContext): boolean {
    const roles = this.reflector.getAllAndOverride<UserRole[]>(ROLES_KEY, [
      context.getHandler(),
      context.getClass(),
    ]);

    if (!roles?.length) {
      return true;
    }

    const request = context.switchToHttp().getRequest<RequestWithUser>();

    if (!request.user) {
      return false;
    }

    return roles.includes(request.user.role);
  }
}
