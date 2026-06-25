import {
  Injectable,
  UnauthorizedException,
  ConflictException,
} from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import * as bcrypt from 'bcrypt';
import { JwtService } from '@nestjs/jwt';
import { User, UserDocument } from './schemas/user.schema';

export interface AuthResponse {
  access_token: string;
}

@Injectable()
export class AuthService {
  constructor(
    @InjectModel(User.name) private userModel: Model<UserDocument>,
    private jwtService: JwtService,
  ) {}

  /**
   * Registers a new user and creates their profile.
   *
   * @param email User email address.
   * @param username Public username.
   * @param pass Plaintext password.
   * @returns A JWT access token.
   */
  async register(
    email: string,
    username: string,
    pass: string,
  ): Promise<AuthResponse> {
    const salt = await bcrypt.genSalt();
    const hashedPassword = await bcrypt.hash(pass, salt);
    const user = new this.userModel({
      email: this.normalizeEmail(email),
      username: this.normalizeUsername(username),
      password: hashedPassword,
    });
    try {
      const savedUser = await user.save();
      return this.createAuthResponse(savedUser);
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
   * @param email User email address.
   * @param pass Plaintext password.
   * @returns A JWT access token.
   */
  async login(email: string, pass: string): Promise<AuthResponse> {
    const user = await this.userModel
      .findOne({ email: this.normalizeEmail(email) })
      .select('+password');
    if (!user || !(await bcrypt.compare(pass, user.password))) {
      throw new UnauthorizedException();
    }
    return this.createAuthResponse(user);
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
