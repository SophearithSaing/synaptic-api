import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { Topic, TopicSchema } from './schemas/topic.schema';
import { TopicsService } from './topics.service';
import { TopicsController } from './topics.controller';
import { RolesGuard } from '../auth/guards/roles.guard';

@Module({
  imports: [
    MongooseModule.forFeature([{ name: Topic.name, schema: TopicSchema }]),
  ],
  providers: [TopicsService, RolesGuard],
  controllers: [TopicsController],
  exports: [TopicsService],
})
export class TopicsModule {}
