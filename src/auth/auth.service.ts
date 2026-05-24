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

@Injectable()
export class AuthService {
  constructor(
    @InjectModel(User.name) private userModel: Model<UserDocument>,
    private studentsService: StudentsService,
    private jwtService: JwtService,
  ) {}

  /**
   * Registers a new user and creates their student profile.
   * @param email User email address.
   * @param pass Plaintext password.
   * @returns The registered user's email.
   */
  async register(email: string, pass: string): Promise<{ email: string }> {
    const salt = await bcrypt.genSalt();
    const hashedPassword = await bcrypt.hash(pass, salt);
    const user = new this.userModel({ email, password: hashedPassword });
    try {
      const savedUser = await user.save();
      await this.studentsService.create(savedUser.id);
      return { email: savedUser.email };
    } catch (error: unknown) {
      if (
        error &&
        typeof error === 'object' &&
        (error as { code: number }).code === 11000
      ) {
        throw new ConflictException('Email already exists');
      }
      throw error;
    }
  }

  /**
   * Authenticates a user and generates a JWT access token.
   * @param email User email address.
   * @param pass Plaintext password.
   * @returns An object containing the access token.
   */
  async login(email: string, pass: string): Promise<{ access_token: string }> {
    const user = await this.userModel.findOne({ email });
    if (!user || !(await bcrypt.compare(pass, user.password))) {
      throw new UnauthorizedException();
    }
    const payload = { email: user.email, sub: user.id };
    return {
      access_token: this.jwtService.sign(payload),
    };
  }
}
