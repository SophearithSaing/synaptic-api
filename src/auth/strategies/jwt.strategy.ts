import { ExtractJwt, Strategy } from 'passport-jwt';
import { PassportStrategy } from '@nestjs/passport';
import { Injectable, UnauthorizedException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import type { Request } from 'express';
import { ACCESS_TOKEN_COOKIE_NAME } from '../auth-cookie';
import { User, UserDocument, UserRole } from '../schemas/user.schema';
import { AuthenticatedUser } from '../types/request-with-user.type';

interface JwtPayload {
  email: string;
  username: string;
  sub: string;
}

interface CookieAuthRequest extends Request {
  cookies: Record<string, string | undefined>;
}

/**
 * Extracts a JWT from the access token cookie.
 *
 * @param request Incoming request.
 * @returns Access token from cookies, or null when unavailable.
 */
function extractJwtFromAccessTokenCookie(
  request: CookieAuthRequest,
): string | null {
  return request.cookies?.[ACCESS_TOKEN_COOKIE_NAME] ?? null;
}

@Injectable()
export class JwtStrategy extends PassportStrategy(Strategy) {
  constructor(
    configService: ConfigService,
    @InjectModel(User.name) private readonly userModel: Model<UserDocument>,
  ) {
    super({
      jwtFromRequest: ExtractJwt.fromExtractors([
        ExtractJwt.fromAuthHeaderAsBearerToken(),
        extractJwtFromAccessTokenCookie,
      ]),
      audience: configService.getOrThrow<string>('JWT_AUDIENCE'),
      ignoreExpiration: false,
      issuer: configService.getOrThrow<string>('JWT_ISSUER'),
      secretOrKey: configService.getOrThrow<string>('JWT_SECRET'),
    });
  }

  /**
   * Validates the JWT payload and returns the current user information.
   *
   * @param payload The decoded JWT payload.
   * @returns The user attached to the request.
   */
  async validate(payload: JwtPayload): Promise<AuthenticatedUser> {
    const user = await this.userModel.findById(payload.sub);

    if (!user) {
      throw new UnauthorizedException();
    }

    return {
      email: user.email,
      username: user.username,
      role: user.role ?? UserRole.User,
      userId: user.id,
    };
  }
}
