import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { HydratedDocument, Types } from 'mongoose';
import type { Question } from '../../questions/schemas/question-set.schema';
import { LiveSession } from './live-session.schema';

export type LiveQuestionDocument = HydratedDocument<LiveQuestion>;

@Schema({ timestamps: true, collection: 'liveQuestions' })
export class LiveQuestion {
  @Prop({ type: Types.ObjectId, ref: LiveSession.name, required: true })
  liveSession: Types.ObjectId;

  @Prop({ type: Object, required: true })
  question: Question;
}

export const LiveQuestionSchema = SchemaFactory.createForClass(LiveQuestion);
