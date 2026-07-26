import { z } from 'zod';
import { QuestionType } from '../../questions/schemas/question-set.schema';
import type { CompletionCreateParams } from 'together-ai/resources/chat.mjs';

export const WRITTEN_EVALUATION_SYSTEM_PROMPT = `You are evaluating written answers for a computing theory learning session. Score each answer from 0 to 1 based on how well it covers the expected key points while avoiding listed misconceptions. Return one evaluation for every submitted answer, using the provided questionId for each result. Include a concise correctAnswer for each question. Keep feedback concise and helpful, and use short concept labels for strengths and weaknesses.`;

export const QUESTION_GENERATION_SYSTEM_PROMPT = `You generate one computing theory learning question for a live session. Return exactly one question matching the requested type. Keep prompts clear and concise. MCQ questions must include three options, one correctOptionId, target concepts, feedback, and rubrics. Written questions must include target concepts, feedback, and rubrics, and may omit options and correctOptionId.`;

export const generatedQuestionSchema = z.object({
  id: z.string(),
  type: z.enum([QuestionType.MCQ, QuestionType.Written]),
  prompt: z.string(),
  options: z
    .array(
      z.object({
        id: z.string(),
        text: z.string(),
      }),
    )
    .optional(),
  correctOptionId: z.string().optional(),
  targetConcepts: z.array(z.string()),
  feedback: z.object({
    correct: z.string(),
    incorrect: z.string(),
  }),
  rubrics: z.object({
    keyPoints: z.array(z.string()),
    misconceptions: z.array(z.string()),
  }),
});

export const generatedLiveQuestionSchema = z.object({
  question: generatedQuestionSchema,
});

export const generatedLiveQuestionResponseFormat: CompletionCreateParams.JsonSchema =
  {
    type: 'json_schema',
    json_schema: {
      name: 'generated_live_question',
      description: 'A generated live-session question.',
      strict: true,
      schema: {
        type: 'object',
        additionalProperties: false,
        properties: {
          question: {
            type: 'object',
            additionalProperties: false,
            properties: {
              id: { type: 'string' },
              type: { type: 'string', enum: ['mcq', 'written'] },
              prompt: { type: 'string' },
              options: {
                type: 'array',
                items: {
                  type: 'object',
                  additionalProperties: false,
                  properties: {
                    id: { type: 'string' },
                    text: { type: 'string' },
                  },
                  required: ['id', 'text'],
                },
              },
              correctOptionId: { type: 'string' },
              targetConcepts: {
                type: 'array',
                items: { type: 'string' },
              },
              feedback: {
                type: 'object',
                additionalProperties: false,
                properties: {
                  correct: { type: 'string' },
                  incorrect: { type: 'string' },
                },
                required: ['correct', 'incorrect'],
              },
              rubrics: {
                type: 'object',
                additionalProperties: false,
                properties: {
                  keyPoints: {
                    type: 'array',
                    items: { type: 'string' },
                  },
                  misconceptions: {
                    type: 'array',
                    items: { type: 'string' },
                  },
                },
                required: ['keyPoints', 'misconceptions'],
              },
            },
            required: [
              'id',
              'type',
              'prompt',
              'targetConcepts',
              'feedback',
              'rubrics',
            ],
          },
        },
        required: ['question'],
      },
    },
  };

export const writtenAnswerEvaluationSchema = z.object({
  questionId: z.string(),
  score: z.number().min(0).max(1),
  correctAnswer: z.string(),
  feedback: z.string(),
  strengths: z.array(z.string()),
  weaknesses: z.array(z.string()),
});

export const writtenAnswerEvaluationsSchema = z.object({
  evaluations: z.array(writtenAnswerEvaluationSchema),
});

export const writtenAnswerEvaluationResponseFormat = {
  type: 'json_schema',
  json_schema: {
    name: 'written_answer_evaluations',
    description: 'Evaluation results for submitted written answers.',
    strict: true,
    schema: {
      type: 'object',
      additionalProperties: false,
      properties: {
        evaluations: {
          type: 'array',
          items: {
            type: 'object',
            additionalProperties: false,
            properties: {
              questionId: { type: 'string' },
              score: {
                type: 'number',
                minimum: 0,
                maximum: 1,
              },
              correctAnswer: { type: 'string' },
              feedback: { type: 'string' },
              strengths: {
                type: 'array',
                items: { type: 'string' },
              },
              weaknesses: {
                type: 'array',
                items: { type: 'string' },
              },
            },
            required: [
              'questionId',
              'score',
              'correctAnswer',
              'feedback',
              'strengths',
              'weaknesses',
            ],
          },
        },
      },
      required: ['evaluations'],
    },
  },
} as const;
