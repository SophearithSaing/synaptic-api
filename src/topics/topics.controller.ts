import { Body, Controller, Get, Param, Post, UseGuards } from '@nestjs/common';
import { TopicsService } from './topics.service';
import { CreateTopicDto } from './dto/create-topic.dto';
import { TopicResponseDto } from './dto/topic-response.dto';
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
  /**
   * Creates a topics controller.
   *
   * @param topicsService The topics service.
   */
  constructor(private topicsService: TopicsService) {}

  /**
   * Creates a new topic.
   *
   * @param body The topic details.
   * @returns The created topic.
   */
  @Post('create')
  @Roles(UserRole.Admin)
  async createTopic(@Body() body: CreateTopicDto): Promise<TopicResponseDto> {
    return this.topicsService.createTopic(body);
  }

  /**
   * Fetches all topics.
   *
   * @returns The available topics.
   */
  @Get()
  async getTopics(): Promise<TopicResponseDto[]> {
    return this.topicsService.getTopics();
  }

  /**
   * Fetches a topic by its ID.
   *
   * @param id The topic ID.
   * @returns The requested topic.
   */
  @Get(':id')
  async getTopicById(
    @Param('id', MongoIdPipe) id: string,
  ): Promise<TopicResponseDto> {
    return this.topicsService.getTopicById(id);
  }
}
