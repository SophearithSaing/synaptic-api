import {
  Injectable,
  UnauthorizedException,
  ConflictException,
} from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { randomBytes } from 'crypto';
import { ConfigService } from '@nestjs/config';
import { Model, Types } from 'mongoose';
import * as bcrypt from 'bcrypt';
import { JwtService } from '@nestjs/jwt';
import { getRefreshTokenMaxAge } from './auth-cookie';
import {
  AuthSession,
  AuthSessionDocument,
} from './schemas/auth-session.schema';
import { User, UserDocument } from './schemas/user.schema';

export interface AuthResponse {
  access_token: string;
}

export interface AuthSessionResponse extends AuthResponse {
  refresh_token: string;
}

@Injectable()
export class AuthService {
  constructor(
    @InjectModel(User.name) private userModel: Model<UserDocument>,
    @InjectModel(AuthSession.name)
    private authSessionModel: Model<AuthSessionDocument>,
    private jwtService: JwtService,
    private configService: ConfigService,
  ) {}

  /**
   * Registers a new user and creates their profile.
   *
   * @param email User email address.
   * @param username Public username.
   * @param pass Plaintext password.
   * @returns An access token and refresh token.
   */
  async register(
    email: string,
    username: string,
    pass: string,
  ): Promise<AuthSessionResponse> {
    const salt = await bcrypt.genSalt();
    const hashedPassword = await bcrypt.hash(pass, salt);
    const user = new this.userModel({
      email: this.normalizeEmail(email),
      username: this.normalizeUsername(username),
      password: hashedPassword,
    });
    try {
      const savedUser = await user.save();
      return this.createSessionAuthResponse(savedUser);
    } catch (error: unknown) {
      if (
        error &&
        typeof error === 'object' &&
        (error as { code: number }).code === 11000
      ) {
        const keyPattern = (error as { keyPattern?: Record<string, number> })
          .keyPattern;
        if (keyPattern?.['username']) {
          throw new ConflictException('Username already exists');
        }
        throw new ConflictException('Email already exists');
      }
      throw error;
    }
  }

  /**
   * Authenticates a user and generates a JWT access token.
   *
   * @param identifier User email address or username.
   * @param pass Plaintext password.
   * @returns An access token and refresh token.
   */
  async login(identifier: string, pass: string): Promise<AuthSessionResponse> {
    const userQuery = this.userModel.findOne(this.createLoginQuery(identifier));

    if (!identifier.includes('@')) {
      userQuery.collation({ locale: 'en', strength: 2 });
    }

    const user = await userQuery.select('+password');
    if (!user || !(await bcrypt.compare(pass, user.password))) {
      throw new UnauthorizedException();
    }
    return this.createSessionAuthResponse(user);
  }

  /**
   * Refreshes an auth session and rotates its refresh token.
   *
   * @param refreshToken Refresh token from the auth cookie.
   * @returns A new access token and refresh token.
   */
  async refresh(refreshToken: string): Promise<AuthSessionResponse> {
    const { sessionId, secret } = this.parseRefreshToken(refreshToken);
    const session = await this.authSessionModel
      .findById(sessionId)
      .select('+refreshTokenHash');

    if (
      !session ||
      session.revokedAt ||
      session.expiresAt.getTime() <= Date.now() ||
      !(await bcrypt.compare(secret, session.refreshTokenHash))
    ) {
      throw new UnauthorizedException();
    }

    const user = await this.userModel.findById(session.userId);

    if (!user) {
      throw new UnauthorizedException();
    }

    const nextSecret = this.createRefreshTokenSecret();
    session.refreshTokenHash = await bcrypt.hash(
      nextSecret,
      await bcrypt.genSalt(),
    );
    session.expiresAt = this.createRefreshTokenExpiry();
    await session.save();

    return {
      ...this.createAuthResponse(user),
      refresh_token: this.serializeRefreshToken(session.id, nextSecret),
    };
  }

  /**
   * Revokes an auth session by refresh token.
   *
   * @param refreshToken Refresh token from the auth cookie.
   */
  async logout(refreshToken?: string): Promise<void> {
    if (!refreshToken) {
      return;
    }

    const { sessionId } = this.parseRefreshToken(refreshToken);
    await this.authSessionModel.findByIdAndUpdate(sessionId, {
      revokedAt: new Date(),
    });
  }

  /**
   * Creates a user lookup query for a login identifier.
   *
   * @param identifier User email address or username.
   * @returns Mongoose query filter for login.
   */
  private createLoginQuery(identifier: string): Record<string, string> {
    if (identifier.includes('@')) {
      return { email: this.normalizeEmail(identifier) };
    }

    return { username: this.normalizeUsername(identifier) };
  }

  /**
   * Normalizes an email for auth lookups and persistence.
   *
   * @param email User email address.
   * @returns Normalized email address.
   */
  private normalizeEmail(email: string): string {
    return email.trim().toLowerCase();
  }

  /**
   * Normalizes a username for persistence.
   *
   * @param username Public username.
   * @returns Trimmed username.
   */
  private normalizeUsername(username: string): string {
    return username.trim();
  }

  /**
   * Creates an access token and persisted refresh session.
   *
   * @param user User document to create tokens for.
   * @returns Access token and refresh token response.
   */
  private async createSessionAuthResponse(
    user: UserDocument,
  ): Promise<AuthSessionResponse> {
    const secret = this.createRefreshTokenSecret();
    const session = new this.authSessionModel({
      expiresAt: this.createRefreshTokenExpiry(),
      refreshTokenHash: await bcrypt.hash(secret, await bcrypt.genSalt()),
      userId: new Types.ObjectId(user._id.toString()),
    });
    const savedSession = await session.save();

    return {
      ...this.createAuthResponse(user),
      refresh_token: this.serializeRefreshToken(savedSession.id, secret),
    };
  }

  /**
   * Creates a random refresh token secret.
   *
   * @returns Base64url-encoded refresh token secret.
   */
  private createRefreshTokenSecret(): string {
    return randomBytes(32).toString('base64url');
  }

  /**
   * Creates the refresh token expiration date.
   *
   * @returns Refresh token expiration date.
   */
  private createRefreshTokenExpiry(): Date {
    return new Date(Date.now() + getRefreshTokenMaxAge(this.configService));
  }

  /**
   * Parses a refresh token into its session id and secret.
   *
   * @param refreshToken Refresh token to parse.
   * @returns Parsed session id and secret.
   */
  private parseRefreshToken(refreshToken: string): {
    sessionId: string;
    secret: string;
  } {
    const [sessionId, secret] = refreshToken.split('.');

    if (!sessionId || !secret) {
      throw new UnauthorizedException();
    }

    return { sessionId, secret };
  }

  /**
   * Serializes a refresh token for the client cookie.
   *
   * @param sessionId Auth session id.
   * @param secret Refresh token secret.
   * @returns Serialized refresh token.
   */
  private serializeRefreshToken(sessionId: string, secret: string): string {
    return `${sessionId}.${secret}`;
  }

  /**
   * Creates a JWT authentication response for a persisted user.
   *
   * @param user User document to encode in the token.
   * @returns A JWT access token response.
   */
  private createAuthResponse(user: UserDocument): AuthResponse {
    const payload = {
      email: user.email,
      username: user.username,
      sub: user._id.toString(),
    };

    return {
      access_token: this.jwtService.sign(payload),
    };
  }
}
