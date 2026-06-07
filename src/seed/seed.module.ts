import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { SeedService } from './seed.service';
import { Topic, TopicSchema } from '../topics/schemas/topic.schema';
import {
  StudentModel,
  StudentModelSchema,
} from '../students/schemas/student-model.schema';
import {
  Category,
  CategorySchema,
} from '../categories/schemas/category.schema';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: Topic.name, schema: TopicSchema },
      { name: Category.name, schema: CategorySchema },
      { name: StudentModel.name, schema: StudentModelSchema },
    ]),
  ],
  providers: [SeedService],
})
export class SeedModule {}
