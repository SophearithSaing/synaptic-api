import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { HydratedDocument, Types } from 'mongoose';
import { LiveQuestion } from '../../sessions/schemas/live-question.schema';

export type AiLogDocument = HydratedDocument<AiLog>;

export enum AiLogOperation {
  QuestionGeneration = 'question-generation',
}

@Schema({ timestamps: true, collection: 'aiLogs' })
export class AiLog {
  @Prop({ required: true, enum: Object.values(AiLogOperation) })
  operation: AiLogOperation;

  @Prop({ required: true })
  aiModel: string;

  @Prop({ required: true })
  prompt: string;

  @Prop({ required: true })
  output: string;

  @Prop({ type: Types.ObjectId, ref: LiveQuestion.name })
  liveQuestion?: Types.ObjectId;

  createdAt: Date;
  updatedAt: Date;
}

export const AiLogSchema = SchemaFactory.createForClass(AiLog);

AiLogSchema.index({ createdAt: -1 });
