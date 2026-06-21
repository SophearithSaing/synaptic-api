import { z } from 'zod';
import { Question } from '../../questions/schemas/question-set.schema';
import { SubmitAnswerItemDto } from '../dtos';
import { writtenAnswerEvaluationSchema } from './sessions-ai.constant';

export type WrittenAnswerEvaluation = z.infer<
  typeof writtenAnswerEvaluationSchema
>;

export type SubmittedWrittenAnswer = {
  submittedAnswer: SubmitAnswerItemDto;
  question: Question;
};
