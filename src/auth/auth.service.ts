import { Injectable, UnauthorizedException } from '@nestjs/common';
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

  async register(email: string, pass: string) {
    const salt = await bcrypt.genSalt();
    const hashedPassword = await bcrypt.hash(pass, salt);
    const user = new this.userModel({ email, password: hashedPassword });
    const savedUser = await user.save();
    await this.studentsService.create(savedUser.id);
    return { email: savedUser.email };
  }

  async login(email: string, pass: string) {
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
