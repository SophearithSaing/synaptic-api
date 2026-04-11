import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document, Types } from 'mongoose';
import { TopicCategory } from './topic-category.schema';

export type TopicDocument = Topic & Document;

@Schema({ timestamps: true })
export class Topic {
  @Prop({ required: true, trim: true })
  title: string;

  @Prop({ required: true, unique: true, index: true, trim: true })
  slug: string;

  @Prop({ required: true })
  description: string;

  @Prop({ type: Types.ObjectId, ref: TopicCategory.name, required: true })
  category: Types.ObjectId;
}

export const TopicSchema = SchemaFactory.createForClass(Topic);
