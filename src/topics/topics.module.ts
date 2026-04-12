import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { Topic, TopicSchema } from './schemas/topic.schema';
import {
  TopicCategory,
  TopicCategorySchema,
} from './schemas/topic-category.schema';
import { TopicsService } from './topics.service';
import { TopicsController } from './topics.controller';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: Topic.name, schema: TopicSchema },
      { name: TopicCategory.name, schema: TopicCategorySchema },
    ]),
  ],
  providers: [TopicsService],
  controllers: [TopicsController],
  exports: [TopicsService],
})
export class TopicsModule {}
