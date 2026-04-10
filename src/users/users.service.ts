import { Injectable } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { User, UserDocument } from './schemas/user.schema';

@Injectable()
export class UsersService {
  constructor(@InjectModel(User.name) private userModel: Model<UserDocument>) {}

  /**
   * Creates a new user with the specified email and hashed password.
   */
  async create(email: string, passwordHash: string): Promise<UserDocument> {
    const newUser = new this.userModel({ email, passwordHash });
    return newUser.save();
  }

  async findByEmail(email: string): Promise<UserDocument | null> {
    return this.userModel.findOne({ email }).exec();
  }

  async findById(id: string): Promise<UserDocument | null> {
    return this.userModel.findById(id).exec();
  }

  async updateRefreshToken(userId: string, hash: string | null): Promise<void> {
    await this.userModel
      .findByIdAndUpdate(userId, { refreshTokenHash: hash })
      .exec();
  }
}
