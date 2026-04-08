import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document, Types } from 'mongoose';
import { Topic } from '../../topics/schemas/topic.schema';
import { Question } from './question.schema';

export type QuestionSetDocument = QuestionSet & Document;

@Schema({ timestamps: true, collection: 'questionSets' })
export class QuestionSet {
  @Prop({ required: true, index: true })
  userId: string;

  @Prop({ type: Types.ObjectId, ref: 'Topic', required: true })
  topic: Topic | Types.ObjectId;

  @Prop({
    type: [{ type: Types.ObjectId, ref: 'Question' }],
    default: [],
  })
  questions: (Question | Types.ObjectId)[];

  @Prop({ required: true, default: 0 })
  score: number;

  @Prop({ required: true, default: 1 })
  difficulty: number;

  @Prop({ type: [String], default: [] })
  weakConcepts: string[];

  @Prop({ type: [String], default: [] })
  strongConcepts: string[];

  @Prop()
  feedback: string;
}

export const QuestionSetSchema = SchemaFactory.createForClass(QuestionSet);
