import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { HydratedDocument, Types } from 'mongoose';
import type { Question } from '../../questions/schemas/question-set.schema';
import type { Answer } from './set-attempt.schemas';
import { LiveSession } from './live-session.schema';

export type LiveQuestionDocument = HydratedDocument<LiveQuestion>;

export enum LiveQuestionStatus {
  Pending = 'pending',
  Rejected = 'rejected',
  Passed = 'passed',
  Failed = 'failed',
}

@Schema({ timestamps: true, collection: 'liveQuestions' })
export class LiveQuestion {
  @Prop({ type: Types.ObjectId, ref: LiveSession.name, required: true })
  liveSession: Types.ObjectId;

  @Prop({ type: Object, required: true })
  question: Question;

  @Prop({ required: true })
  level: number;

  @Prop({ required: true })
  questionNumber: number;

  @Prop({ required: true, enum: Object.values(LiveQuestionStatus) })
  status: LiveQuestionStatus;

  @Prop({ type: Object })
  answer: Answer;

  @Prop()
  answeredAt: Date;
}

export const LiveQuestionSchema = SchemaFactory.createForClass(LiveQuestion);
