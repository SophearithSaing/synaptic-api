import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { HydratedDocument, Types } from 'mongoose';
import { User } from '../../auth/schemas/user.schema';
import { Session } from '@nestjs/common';
import { Topic } from '../../topics/schemas/topic.schema';
import {
  QuestionSet,
  QuestionType,
} from '../../questions/schemas/question-set.schema';

export type SetAttemptDocument = HydratedDocument<SetAttempt>;

@Schema({ timestamps: true, collection: 'setAttempts' })
export class SetAttempt {
  @Prop({ type: Types.ObjectId, ref: User.name, required: true })
  user: Types.ObjectId;

  @Prop({ type: Types.ObjectId, ref: Session.name, required: true })
  session: Types.ObjectId;

  @Prop({ type: Types.ObjectId, ref: Topic.name, required: true })
  topic: Types.ObjectId;

  @Prop({ type: Types.ObjectId, ref: QuestionSet.name, required: true })
  questionSet: Types.ObjectId;

  @Prop()
  level: number;

  @Prop({ type: [Object] })
  answers: Answer[];

  @Prop()
  setScore: number;

  @Prop()
  passed: boolean;

  @Prop({ type: [String] })
  strength: string[];

  @Prop({ type: [String] })
  weakness: string[];

  @Prop()
  aiSummary: string;

  @Prop()
  submittedAt: Date;

  @Prop()
  evaluatedAt: Date;

  createdAt: Date;
  updatedAt: Date;
}

export const SetAttemptSchema = SchemaFactory.createForClass(SetAttempt);

export interface Answer {
  id: string;
  questionId: string;
  questionType: QuestionType;
  answer: string;
  correctAnswer: string;
  score: number;
  feedback: string;
  targetConcepts: string[];
  strength: string[];
  weakness: string[];
  evaluatedBy: EvaluatedBy;
}

export enum EvaluatedBy {
  System = 'system',
  AI = 'ai',
}
