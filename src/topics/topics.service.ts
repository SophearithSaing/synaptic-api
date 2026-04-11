import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { Topic, TopicDocument } from './schemas/topic.schema';
import {
  TopicCategory,
  TopicCategoryDocument,
} from './schemas/topic-category.schema';
import { CreateCategoryDto, CreateTopicDto } from './dto/create-topic.dto';

@Injectable()
export class TopicsService {
  constructor(
    @InjectModel(Topic.name) private topicModel: Model<TopicDocument>,
    @InjectModel(TopicCategory.name)
    private categoryModel: Model<TopicCategoryDocument>,
  ) {}

  /**
   * Creates a new topic category.
   * @param dto The category details.
   */
  async createCategory(dto: CreateCategoryDto) {
    const category = new this.categoryModel(dto);
    return category.save();
  }

  /**
   * Creates a new topic within a category.
   * @param dto The topic details
   */
  async createTopic(dto: CreateTopicDto) {
    const topic = new this.topicModel(dto);
    return topic.save();
  }

  /**
   * Fetches a topic by its unique ID.
   * @param id The topic ID.
   */
  async getTopicById(id: string) {
    const topic = await this.topicModel
      .findById(id)
      .populate('category')
      .exec();
    if (!topic) {
      throw new NotFoundException('Topic not found');
    }
    return topic;
  }
}
