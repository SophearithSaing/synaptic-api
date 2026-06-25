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
  @Prop({ required: true, trim: true })
  username: string;

  @Prop({ required: true, lowercase: true, trim: true })
  email: string;

  @Prop({ required: true, select: false })
  password: string;

  @Prop({ enum: Object.values(UserRole), default: UserRole.User })
  role: UserRole;
}

export const UserSchema = SchemaFactory.createForClass(User);

UserSchema.index(
  { username: 1 },
  { unique: true, collation: { locale: 'en', strength: 2 } },
);
UserSchema.index({ email: 1 }, { unique: true });
