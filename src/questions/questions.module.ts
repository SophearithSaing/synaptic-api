import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { QuestionSet, QuestionSetSchema } from './schemas/question-set.schema';
import { QuestionsController } from './questions.controller';
import { QuestionsService } from './questions.service';
import { RolesGuard } from '../auth/guards/roles.guard';
import { Topic, TopicSchema } from '../topics/schemas/topic.schema';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: QuestionSet.name, schema: QuestionSetSchema },
      { name: Topic.name, schema: TopicSchema },
    ]),
  ],
  controllers: [QuestionsController],
  providers: [QuestionsService, RolesGuard],
  exports: [QuestionsService],
})
export class QuestionsModule {}
