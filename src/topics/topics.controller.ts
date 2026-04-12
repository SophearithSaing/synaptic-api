import { Controller, Post, Body, Get, Param, UseGuards } from '@nestjs/common';
import { TopicsService } from './topics.service';
import { CreateCategoryDto, CreateTopicDto } from './dto/create-topic.dto';
import { TopicDocument } from './schemas/topic.schema';
import { TopicCategoryDocument } from './schemas/topic-category.schema';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';

/**
 * Controller for managing topics and categories.
 */
@Controller('topics')
@UseGuards(JwtAuthGuard)
export class TopicsController {
  constructor(private topicsService: TopicsService) {}

  /**
   * Creates a new topic category.
   * @param body The category details.
   */
  @Post('category/create')
  async createCategory(
    @Body() body: CreateCategoryDto,
  ): Promise<TopicCategoryDocument> {
    return this.topicsService.createCategory(body);
  }

  /**
   * Creates a new topic.
   * @param body The topic details.
   */
  @Post('create')
  async createTopic(@Body() body: CreateTopicDto): Promise<TopicDocument> {
    return this.topicsService.createTopic(body);
  }

  /**
   * Fetches a topic by its ID.
   * @param id The topic ID.
   */
  @Get(':id')
  async getTopicById(@Param('id') id: string): Promise<TopicDocument> {
    return this.topicsService.getTopicById(id);
  }
}
