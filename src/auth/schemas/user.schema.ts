import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document } from 'mongoose';

export type UserDocument = User & Document;

/**
 * Supported application user roles.
 */
export enum UserRole {
  User = 'user',
  Admin = 'admin',
}

@Schema({ timestamps: true })
export class User {
  @Prop({ required: true, unique: true, index: true, trim: true })
  username: string;

  @Prop({ required: true, unique: true, index: true })
  email: string;

  @Prop({ required: true })
  password: string;

  @Prop({ enum: Object.values(UserRole), default: UserRole.User })
  role: UserRole;
}

export const UserSchema = SchemaFactory.createForClass(User);
