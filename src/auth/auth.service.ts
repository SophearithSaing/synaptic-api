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
import { StudentsService } from '../students/students.service';

export interface AuthResponse {
  access_token: string;
}

@Injectable()
export class AuthService {
  constructor(
    @InjectModel(User.name) private userModel: Model<UserDocument>,
    private studentsService: StudentsService,
    private jwtService: JwtService,
  ) {}

  /**
   * Registers a new user and creates their student profile.
   * @param {string} email - User email address.
   * @param {string} username - Public username.
   * @param {string} pass - Plaintext password.
   * @returns {Promise<AuthResponse>} A JWT access token.
   */
  async register(
    email: string,
    username: string,
    pass: string,
  ): Promise<AuthResponse> {
    const salt = await bcrypt.genSalt();
    const hashedPassword = await bcrypt.hash(pass, salt);
    const user = new this.userModel({
      email,
      username,
      password: hashedPassword,
    });
    try {
      const savedUser = await user.save();
      await this.studentsService.create(savedUser.id);
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
   * @param {string} email - User email address.
   * @param {string} pass - Plaintext password.
   * @returns {Promise<AuthResponse>} A JWT access token.
   */
  async login(email: string, pass: string): Promise<AuthResponse> {
    const user = await this.userModel.findOne({ email });
    if (!user || !(await bcrypt.compare(pass, user.password))) {
      throw new UnauthorizedException();
    }
    return this.createAuthResponse(user);
  }

  /**
   * Creates a JWT authentication response for a persisted user.
   * @param {UserDocument} user - User document to encode in the token.
   * @returns {AuthResponse} A JWT access token response.
   */
  private createAuthResponse(user: UserDocument): AuthResponse {
    const payload = {
      email: user.email,
      username: user.username,
      sub: user._id,
    };

    return {
      access_token: this.jwtService.sign(payload),
    };
  }
}
