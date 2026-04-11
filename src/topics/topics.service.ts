import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { Topic, TopicDocument } from './schemas/topic.schema';
import {
  TopicCategory,
  TopicCategoryDocument,
} from './schemas/topic-category.schema';

@Injectable()
export class TopicsService {
  constructor(
    @InjectModel(Topic.name) private topicModel: Model<TopicDocument>,
    @InjectModel(TopicCategory.name)
    private categoryModel: Model<TopicCategoryDocument>,
  ) {}

  async createCategory(data: {
    title: string;
    slug: string;
    description: string;
  }) {
    const category = new this.categoryModel(data);
    return category.save();
  }

  async createTopic(data: {
    title: string;
    slug: string;
    description: string;
    categoryId: string;
  }) {
    const topic = new this.topicModel(data);
    return topic.save();
  }

  async getTopicById(id: string) {
    const topic = await this.topicModel.findById(id).exec();
    if (!topic) {
      throw new NotFoundException('Topic not found');
    }
    return topic;
  }
}
