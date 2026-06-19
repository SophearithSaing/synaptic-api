import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document, Types } from 'mongoose';
import { User } from '../../auth/schemas/user.schema';
import { Topic } from '../../topics/schemas/topic.schema';

export type SessionDocument = Session & Document;

@Schema({ timestamps: true, collection: 'sessions' })
export class Session {
  @Prop({ type: Types.ObjectId, ref: User.name, required: true })
  student: Types.ObjectId;

  @Prop({ type: Types.ObjectId, ref: Topic.name, required: true })
  topic: Types.ObjectId;

  @Prop()
  currentLevel: number;

  @Prop()
  status: string;

  @Prop({ type: Object })
  overallEvaluation: OverallEvaluation;

  @Prop()
  startAt: Date;

  @Prop()
  finishAt: Date;

  createdAt: Date;
  updatedAt: Date;
}

export const SessionSchema = SchemaFactory.createForClass(Session);

export interface OverallEvaluation {
  summary: string;
  stengths: string[];
  weakness: string[];
  recommendations: string[];
}
