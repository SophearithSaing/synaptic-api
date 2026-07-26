import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { HydratedDocument, Types } from 'mongoose';
import { User } from '../../auth/schemas/user.schema';
import { Topic } from '../../topics/schemas/topic.schema';
import { SessionStatus } from './session.schema';
import type { OverallEvaluation } from './session.schema';

export type LiveSessionDocument = HydratedDocument<LiveSession>;

@Schema({ timestamps: true, collection: 'liveSessions' })
export class LiveSession {
  @Prop({ type: Types.ObjectId, ref: User.name, required: true })
  student: Types.ObjectId;

  @Prop({ type: Types.ObjectId, ref: Topic.name, required: true })
  topic: Types.ObjectId;

  @Prop()
  currentLevel: number;

  @Prop({ type: String, enum: Object.values(SessionStatus) })
  status: SessionStatus;

  @Prop({ type: Object })
  overallEvaluation: OverallEvaluation;

  @Prop()
  startedAt: Date;

  @Prop()
  finishedAt: Date;

  createdAt: Date;
  updatedAt: Date;
}

export const LiveSessionSchema = SchemaFactory.createForClass(LiveSession);
