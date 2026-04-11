import { Controller, Post, Body, Get, Param } from '@nestjs/common';
import { TopicsService } from './topics.service';
import { CreateCategoryDto, CreateTopicDto } from './dto/create-topic.dto';

@Controller('topics')
export class TopicsController {
  constructor(private topicsService: TopicsService) {}

  /**
   * Creates a new topic category.
   * @param body The category details.
   */
  @Post('category/create')
  async createCategory(@Body() body: CreateCategoryDto) {
    return this.topicsService.createCategory(body);
  }

  /**
   * Creates a new topic.
   * @param body The topic details.
   */
  @Post('create')
  async createTopic(@Body() body: CreateTopicDto) {
    return this.topicsService.createTopic(body);
  }

  /**
   * Fetches a topic by its ID.
   * @param id The topic ID.
   */
  @Get(':id')
  async getTopicById(@Param('id') id: string) {
    return this.topicsService.getTopicById(id);
  }
}
