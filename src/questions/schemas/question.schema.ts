import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document, Schema as MongooseSchema } from 'mongoose';

export enum QuestionType {
  Mcq = 'mcq',
  Written = 'written',
}

export type QuestionDocument = Question & Document;

@Schema({ timestamps: true })
export class Question {
  @Prop({
    required: true,
    enum: QuestionType,
  })
  type: QuestionType;

  @Prop({ required: true })
  text: string;

  @Prop({ type: [String], default: [] })
  options: string[];

  @Prop()
  correctOption: string;

  @Prop({ type: [String], default: [] })
  idealAnswerPoints: string[];

  @Prop()
  studentAnswer: string;

  @Prop()
  isCorrect: boolean;

  @Prop({ default: 0 })
  score: number;

  @Prop()
  feedback: string;

  @Prop({
    required: true,
  })
  difficulty: number;
}

export const QuestionSchema = SchemaFactory.createForClass(Question);
