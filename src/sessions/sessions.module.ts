import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { SessionsController } from './sessions.controller';
import { SessionsService } from './sessions.service';
import {
  QuestionSet,
  QuestionSetSchema,
} from '../questions/schemas/question-set.schema';
import { Topic, TopicSchema } from '../topics/schemas/topic.schema';
import { AiModule } from '../ai/ai.module';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: QuestionSet.name, schema: QuestionSetSchema },
      { name: Topic.name, schema: TopicSchema },
    ]),
    AiModule,
  ],
  controllers: [SessionsController],
  providers: [SessionsService],
})
export class SessionsModule {}
