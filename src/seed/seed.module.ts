import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { SeedService } from './seed.service';
import { Topic, TopicSchema } from '../topics/schemas/topic.schema';
import {
  TopicCategory,
  TopicCategorySchema,
} from '../topics/schemas/topic-category.schema';
import {
  StudentModel,
  StudentModelSchema,
} from '../students/schemas/student-model.schema';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: Topic.name, schema: TopicSchema },
      { name: TopicCategory.name, schema: TopicCategorySchema },
      { name: StudentModel.name, schema: StudentModelSchema },
    ]),
  ],
  providers: [SeedService],
})
export class SeedModule {}
