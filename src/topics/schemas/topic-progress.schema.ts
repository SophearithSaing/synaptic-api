import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document, Schema as MongooseSchema } from 'mongoose';
import { Topic } from './topic.schema';
import { QuestionSet } from '../../questions/schemas/question-set.schema';

export enum TopicStatus {
  InProgress = 'in-progress',
  Completed = 'completed',
}

export type TopicProgressDocument = TopicProgress & Document;

@Schema({ timestamps: true, collection: 'topicProgresses' })
export class TopicProgress {
  @Prop({ required: true, index: true })
  userId: string;

  @Prop({ type: MongooseSchema.Types.ObjectId, ref: 'Topic', required: true })
  topic: Topic;

  @Prop({
    type: [{ type: MongooseSchema.Types.ObjectId, ref: 'QuestionSet' }],
    default: [],
  })
  history: QuestionSet[];

  @Prop({ required: true, default: 0 })
  currentLevel: number;

  @Prop({
    required: true,
    enum: TopicStatus,
  })
  status: TopicStatus;
}

export const TopicProgressSchema = SchemaFactory.createForClass(TopicProgress);
