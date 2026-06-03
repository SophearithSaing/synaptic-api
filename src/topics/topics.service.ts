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
   * @returns The created topic category.
   */
  async createCategory(dto: CreateCategoryDto): Promise<TopicCategoryDocument> {
    const category = new this.categoryModel(dto);
    return category.save();
  }

  /**
   * Creates a new topic within a category.
   * @param dto The topic details.
   * @returns The created topic.
   */
  async createTopic(dto: CreateTopicDto): Promise<TopicDocument> {
    const topic = new this.topicModel(dto);
    return topic.save();
  }

  /**
   * Fetches all topic categories.
   * @returns The topic categories sorted by title.
   */
  async getCategories(): Promise<TopicCategoryDocument[]> {
    return this.categoryModel.find().sort({ title: 1 }).exec();
  }

  /**
   * Fetches all topics.
   * @returns The topics sorted by title with categories populated.
   */
  async getTopics(): Promise<TopicDocument[]> {
    return this.topicModel
      .find()
      .sort({ title: 1 })
      .populate('category')
      .exec();
  }

  /**
   * Fetches a topic by its unique ID.
   * @param id The topic ID.
   * @returns The requested topic.
   */
  async getTopicById(id: string): Promise<TopicDocument> {
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
