import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document } from 'mongoose';

export type TopicCategoryDocument = TopicCategory & Document;

@Schema({ timestamps: true })
export class TopicCategory {
  @Prop({ required: true })
  title: string;

  @Prop({ required: true, unique: true, index: true })
  slug: string;

  @Prop({ required: true })
  description: string;
}

export const TopicCategorySchema = SchemaFactory.createForClass(TopicCategory);
