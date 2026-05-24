import { SetMetadata } from '@nestjs/common';
import { UserRole } from '../schemas/user.schema';

export const ROLES_KEY = 'roles';

/**
 * Marks a route as requiring one of the provided roles.
 * @param roles Roles allowed to access the route.
 * @returns Nest metadata decorator for role requirements.
 */
export function Roles(...roles: UserRole[]): ReturnType<typeof SetMetadata> {
  return SetMetadata(ROLES_KEY, roles);
}
