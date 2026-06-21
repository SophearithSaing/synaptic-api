import { z } from 'zod';

export const WRITTEN_EVALUATION_SYSTEM_PROMPT = `You are evaluating written answers for a computing theory learning session. Score each answer from 0 to 1 based on how well it covers the expected key points while avoiding listed misconceptions. Return one evaluation for every submitted answer, using the provided questionId for each result. Keep feedback concise and helpful, and use short concept labels for strength and weakness.`;

export const writtenAnswerEvaluationSchema = z.object({
  questionId: z.string(),
  score: z.number().min(0).max(1),
  feedback: z.string(),
  strength: z.array(z.string()),
  weakness: z.array(z.string()),
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
              feedback: { type: 'string' },
              strength: {
                type: 'array',
                items: { type: 'string' },
              },
              weakness: {
                type: 'array',
                items: { type: 'string' },
              },
            },
            required: [
              'questionId',
              'score',
              'feedback',
              'strength',
              'weakness',
            ],
          },
        },
      },
      required: ['evaluations'],
    },
  },
} as const;
