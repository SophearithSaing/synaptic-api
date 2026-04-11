import { Controller, Post, Body, Get, Param } from '@nestjs/common';
import { TopicsService } from './topics.service';
import { CreateCategoryDto, CreateTopicDto } from './dto/create-topic.dto';

@Controller('topics')
export class TopicsController {
  constructor(private topicsService: TopicsService) {}

  @Post('category/create')
  async createCategory(@Body() body: CreateCategoryDto) {
    return this.topicsService.createCategory(body);
  }

  @Post('create')
  async createTopic(@Body() body: CreateTopicDto) {
    return this.topicsService.createTopic(body);
  }

  @Get(':id')
  async getTopicById(@Param('id') id: string) {
    return this.topicsService.getTopicById(id);
  }
}
