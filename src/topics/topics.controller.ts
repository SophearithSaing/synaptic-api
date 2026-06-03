import { Body, Controller, Get, Param, Post, UseGuards } from '@nestjs/common';
import { TopicsService } from './topics.service';
import { CreateCategoryDto, CreateTopicDto } from './dto/create-topic.dto';
import { TopicDocument } from './schemas/topic.schema';
import { TopicCategoryDocument } from './schemas/topic-category.schema';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import { RolesGuard } from '../auth/guards/roles.guard';
import { Roles } from '../auth/decorators/roles.decorator';
import { UserRole } from '../auth/schemas/user.schema';
import { MongoIdPipe } from '../common/pipes/mongo-id.pipe';

/**
 * Controller for managing topics and categories.
 */
@Controller('topics')
@UseGuards(JwtAuthGuard, RolesGuard)
export class TopicsController {
  constructor(private topicsService: TopicsService) {}

  /**
   * Creates a new topic category.
   * @param body The category details.
   * @returns The created topic category.
   */
  @Post('category/create')
  @Roles(UserRole.Admin)
  async createCategory(
    @Body() body: CreateCategoryDto,
  ): Promise<TopicCategoryDocument> {
    return this.topicsService.createCategory(body);
  }

  /**
   * Creates a new topic.
   * @param body The topic details.
   * @returns The created topic.
   */
  @Post('create')
  @Roles(UserRole.Admin)
  async createTopic(@Body() body: CreateTopicDto): Promise<TopicDocument> {
    return this.topicsService.createTopic(body);
  }

  /**
   * Fetches all topics.
   * @returns The available topics.
   */
  @Get()
  async getTopics(): Promise<TopicDocument[]> {
    return this.topicsService.getTopics();
  }

  /**
   * Fetches all topic categories.
   * @returns The available topic categories.
   */
  @Get('categories')
  async getCategories(): Promise<TopicCategoryDocument[]> {
    return this.topicsService.getCategories();
  }

  /**
   * Fetches a topic by its ID.
   * @param id The topic ID.
   * @returns The requested topic.
   */
  @Get(':id')
  async getTopicById(
    @Param('id', MongoIdPipe) id: string,
  ): Promise<TopicDocument> {
    return this.topicsService.getTopicById(id);
  }
}
