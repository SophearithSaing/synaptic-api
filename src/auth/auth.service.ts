import {
  Injectable,
  ConflictException,
  ForbiddenException,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { ConfigService } from '@nestjs/config';
import * as bcrypt from 'bcrypt';
import { UsersService } from '../users/users.service';

export interface JwtPayload {
  email: string;
  sub: string;
}

export interface RefreshPayload {
  sub: string;
}

export interface UserObject {
  _id: string;
  email: string;
  passwordHash?: string;
}

@Injectable()
export class AuthService {
  constructor(
    private usersService: UsersService,
    private jwtService: JwtService,
    private configService: ConfigService,
  ) {}

  /**
   * Validates a user based on email and password.
   */
  async validateUser(email: string, pass: string): Promise<UserObject | null> {
    const user = await this.usersService.findByEmail(email);
    if (user && (await bcrypt.compare(pass, user.passwordHash))) {
      const userObj = user.toObject() as UserObject;
      const { passwordHash, ...result } = userObj;
      void passwordHash;
      return result as UserObject;
    }
    return null;
  }

  /**
   * Generates new access and refresh tokens for a user.
   */
  async login(user: UserObject) {
    const payload: JwtPayload = { email: user.email, sub: user._id };
    const accessToken = this.jwtService.sign(payload, {
      secret: this.configService.get<string>('JWT_SECRET'),
      expiresIn: '15m',
    });
    const refreshToken = this.jwtService.sign(
      { sub: user._id },
      {
        secret: this.configService.get<string>('JWT_REFRESH_SECRET'),
        expiresIn: '7d',
      },
    );

    const refreshTokenHash = await bcrypt.hash(refreshToken, 10);
    await this.usersService.updateRefreshToken(
      user._id.toString(),
      refreshTokenHash,
    );

    return {
      accessToken,
      refreshToken,
    };
  }

  /**
   * Registers a new user.
   */
  async register(email: string, pass: string) {
    const existingUser = await this.usersService.findByEmail(email);
    if (existingUser) {
      throw new ConflictException('Email already exists');
    }

    const passwordHash = await bcrypt.hash(pass, 10);
    await this.usersService.create(email, passwordHash);

    return { message: 'User registered successfully' };
  }

  /**
   * Refreshes access and refresh tokens.
   */
  async refresh(refreshToken: string) {
    // 1. Verify the token signature first
    let payload: RefreshPayload;
    try {
      payload = this.jwtService.verify(refreshToken, {
        secret: this.configService.getOrThrow<string>('JWT_REFRESH_SECRET'),
      }) as RefreshPayload;
    } catch {
      throw new ForbiddenException('Expired or invalid refresh token');
    }

    if (!payload || !payload.sub) {
      throw new ForbiddenException('Invalid refresh token');
    }

    // 2. Perform DB lookup only after verification
    const user = await this.usersService.findById(payload.sub);
    if (!user || !user.refreshTokenHash) {
      throw new ForbiddenException('Invalid refresh token');
    }

    const isMatch = await bcrypt.compare(refreshToken, user.refreshTokenHash);
    if (!isMatch) {
      throw new ForbiddenException('Invalid refresh token');
    }

    const userObj = user.toObject() as UserObject;
    return this.login(userObj);
  }

  /**
   * Invalidates the user refresh token.
   */
  async logout(userId: string) {
    await this.usersService.updateRefreshToken(userId, null);
    return { message: 'Logged out successfully' };
  }
}
