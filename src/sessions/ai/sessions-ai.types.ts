import { z } from 'zod';
import { Types } from 'mongoose';
import {
  Question,
  QuestionType,
} from '../../questions/schemas/question-set.schema';
import { SubmitAnswerItemDto } from '../dtos';
import { writtenAnswerEvaluationSchema } from './sessions-ai.constant';

export type WrittenAnswerEvaluation = z.infer<
  typeof writtenAnswerEvaluationSchema
>;

export type SubmittedWrittenAnswer = {
  submittedAnswer: SubmitAnswerItemDto;
  question: Question;
};

export type GeneratedLiveQuestion = {
  question: Question;
};

export type GeneratedLiveQuestionResult = {
  question: Question;
  aiLogId: Types.ObjectId;
};

export type RecentAcceptedQuestionContext = {
  level: number;
  prompt: string;
  targetConcepts: string[];
};

export type LiveGenerationPromptContext = {
  topicSlug: string;
  topicTitle: string;
  topicDescription: string;
  topicTags: string[];
  level: number;
  questionNumber: number;
  questionType: QuestionType;
  recentAcceptedQuestions: RecentAcceptedQuestionContext[];
  rejectedQuestion?: Question;
  rejectionReason?: string;
};
