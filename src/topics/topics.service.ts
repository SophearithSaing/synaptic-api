import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { Topic, TopicDocument } from './schemas/topic.schema';
import { CreateTopicDto } from './dto/create-topic.dto';
import { TopicResponseDto } from './dto/topic-response.dto';
import { CategoriesService } from '../categories/categories.service';
import { CategoryDocument } from '../categories/schemas/category.schema';

@Injectable()
export class TopicsService {
  constructor(
    @InjectModel(Topic.name) private topicModel: Model<TopicDocument>,
  ) {}

  /**
   * Creates a new topic within a category.
   *
   * @param dto The topic details.
   * @returns The created topic.
   */
  async createTopic(dto: CreateTopicDto): Promise<TopicResponseDto> {
    const topic = new this.topicModel(dto);
    const savedTopic = await topic.save();
    await savedTopic.populate('category');

    return this.toResponse(savedTopic);
  }

  /**
   * Fetches all topics.
   *
   * @returns The topics sorted by title.
   */
  async getTopics(): Promise<TopicResponseDto[]> {
    const topics = await this.topicModel
      .find()
      .sort({ title: 1 })
      .populate('category')
      .exec();

    return topics.map((topic) => this.toResponse(topic));
  }

  /**
   * Fetches a topic by its unique ID.
   *
   * @param id The topic ID.
   * @returns The requested topic.
   */
  async getTopicById(id: string): Promise<TopicResponseDto> {
    const topic = await this.topicModel
      .findById(id)
      .populate('category')
      .exec();

    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    return this.toResponse(topic);
  }

  /**
   * Converts a topic document to an API response.
   *
   * @param topic The topic document.
   * @returns The topic response.
   */
  private toResponse(topic: TopicDocument): TopicResponseDto {
    return {
      id: topic._id.toString(),
      title: topic.title,
      slug: topic.slug,
      description: topic.description,
      icon: topic.icon,
      tags: topic.tags,
      category: CategoriesService.toResponse(
        topic.category as unknown as CategoryDocument,
      ),
    };
  }
}
