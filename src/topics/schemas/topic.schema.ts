import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { HydratedDocument, Types } from 'mongoose';
import { Category } from '../../categories/schemas/category.schema';

export type TopicDocument = HydratedDocument<Topic>;

@Schema({ timestamps: true, collection: 'topics' })
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

  @Prop({ type: Types.ObjectId, ref: Category.name, required: true })
  category: Types.ObjectId;
}

/**
 * Validates the number of topic tags.
 *
 * @param tags The topic tags.
 * @returns Whether the tag count is valid.
 */
function tagsLimit(tags: string[]): boolean {
  return Array.isArray(tags) && tags.length >= 1 && tags.length <= 2;
}

export const TopicSchema = SchemaFactory.createForClass(Topic);
