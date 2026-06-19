import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document, Types } from 'mongoose';

export type QuestionSetDocument = QuestionSet & Document;

@Schema({ timestamps: true, collection: 'questionSets' })
export class QuestionSet {
  @Prop({ type: Types.ObjectId, ref: 'Topic', required: true })
  topic: Types.ObjectId;

  @Prop({ required: true })
  setType: string;

  @Prop({ required: true })
  level: number;

  @Prop({ type: [Object], required: true })
  questions: Question[];
}

export const QuestionSetSchema = SchemaFactory.createForClass(QuestionSet);

export interface Question {
  id: string;
  type: string;
  prompt: string;
  options: Array<{ id: string; text: string }>;
  correctionOptionId: string;
  targetConcepts: string[];
  feedback: { correct: string; incorrect: string };
  rubric: { keyPoints: string[]; misconceptions: string[] };
}
