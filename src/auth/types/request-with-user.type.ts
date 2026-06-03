import { Request } from 'express';
import { UserRole } from '../schemas/user.schema';

/**
 * Authenticated user attached by the JWT strategy.
 */
export interface AuthenticatedUser {
  email: string;
  username: string;
  userId: string;
  role: UserRole;
}

/**
 * Express request containing the authenticated user.
 */
export interface RequestWithUser extends Request {
  user: AuthenticatedUser;
}
