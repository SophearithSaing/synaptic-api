import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { HydratedDocument, Types } from 'mongoose';
import { User } from '../../auth/schemas/user.schema';
import { Session } from './session.schema';
import { Topic } from '../../topics/schemas/topic.schema';

export type SessionEvaluationDocument = HydratedDocument<SessionEvaluation>;

@Schema({ timestamps: true, collection: 'sessionEvaluations' })
export class SessionEvaluation {
  @Prop({ type: Types.ObjectId, ref: User.name, required: true })
  student: Types.ObjectId;

  @Prop({ type: Types.ObjectId, ref: Session.name, required: true })
  session: Types.ObjectId;

  @Prop({ type: Types.ObjectId, ref: Topic.name, required: true })
  topic: Types.ObjectId;

  @Prop()
  fromLevel: number;

  @Prop()
  toLevel: number;

  @Prop()
  overallScore: number;

  @Prop()
  summary: string;

  @Prop({ type: [String] })
  stength: string[];

  @Prop({ type: [String] })
  weakness: string[];

  @Prop({ type: [String] })
  recommendation: string[];

  @Prop({ type: [String] })
  attemptIds: string[];

  createdAt: Date;
  updatedAt: Date;
}

export const SessionEvaluationSchema =
  SchemaFactory.createForClass(SessionEvaluation);
