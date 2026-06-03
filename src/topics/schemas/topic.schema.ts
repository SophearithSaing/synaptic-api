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

  @Prop({ required: true })
  icon: string;

  @Prop({
    type: [String],
    required: true,
    validate: [tagsLimit, 'Topic tags must include 1 or 2 items.'],
  })
  tags: string[];

  @Prop({ type: Types.ObjectId, ref: TopicCategory.name, required: true })
  category: Types.ObjectId;
}

/**
 * Validates the number of topic tags.
 * @param {string[]} tags - The topic tags.
 * @returns {boolean}
 */
function tagsLimit(tags: string[]): boolean {
  return Array.isArray(tags) && tags.length >= 1 && tags.length <= 2;
}

export const TopicSchema = SchemaFactory.createForClass(Topic);
