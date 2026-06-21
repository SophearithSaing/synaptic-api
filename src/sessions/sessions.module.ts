import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { SessionsController } from './sessions.controller';
import { SessionsService } from './sessions.service';
import {
  QuestionSet,
  QuestionSetSchema,
} from '../questions/schemas/question-set.schema';
import { Topic, TopicSchema } from '../topics/schemas/topic.schema';
import {
  SessionEvaluation,
  SessionEvaluationSchema,
} from './schemas/session-evaluation.schemas';
import { Session, SessionSchema } from './schemas/session.schema';
import { SetAttempt, SetAttemptSchema } from './schemas/set-attempt.schemas';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: QuestionSet.name, schema: QuestionSetSchema },
      { name: Session.name, schema: SessionSchema },
      { name: SessionEvaluation.name, schema: SessionEvaluationSchema },
      { name: SetAttempt.name, schema: SetAttemptSchema },
      { name: Topic.name, schema: TopicSchema },
    ]),
  ],
  controllers: [SessionsController],
  providers: [SessionsService],
})
export class SessionsModule {}
