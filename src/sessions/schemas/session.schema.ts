import { Prop, SchemaFactory } from '@nestjs/mongoose';
import { Document, Types } from 'mongoose';
import { User } from '../../auth/schemas/user.schema';
import { Topic } from '../../topics/schemas/topic.schema';

export type SessionDocument = Session & Document;

export class Session {
  @Prop({ type: Types.ObjectId, ref: User.name, required: true })
  student: Types.ObjectId;

  @Prop({ type: Types.ObjectId, ref: Topic.name, required: true })
  topic: Types.ObjectId;

  @Prop()
  currentLevel: number;

  @Prop()
  status: string;

  @Prop()
  overallEvaluation: overallEvaluation;

  @Prop()
  startAt: Date;

  @Prop()
  finishAt: Date;
}

export const SessionSchema = SchemaFactory.createForClass(Session);

export interface overallEvaluation {
  summary: string;
  stengths: string[];
  weakness: string[];
  recommendations: string[];
}
