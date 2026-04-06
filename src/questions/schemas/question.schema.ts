import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document, Schema as MongooseSchema } from 'mongoose';

export enum QuestionType {
  MCQ = 'MCQ',
  WRITTEN = 'WRITTEN',
}

export type QuestionDocument = Question & Document;

@Schema({ timestamps: true })
export class Question {
  @Prop({
    required: true,
    enum: QuestionType,
  })
  type: QuestionType;

  @Prop({
    type: MongooseSchema.Types.Mixed,
    required: true,
  })
  metadata: any;

  @Prop({
    required: true,
  })
  difficulty: number;
}

export const QuestionSchema = SchemaFactory.createForClass(Question);
