import { ServiceUnavailableException } from '@nestjs/common';
import { SessionEvaluationDocument } from './schemas/session-evaluation.schemas';
import { Answer, SetAttemptDocument } from './schemas/set-attempt.schemas';
import {
  Question,
  QuestionType,
} from '../questions/schemas/question-set.schema';
import { generatedLiveQuestionSchema } from './ai/sessions-ai.constant';
import {
  GeneratedLiveQuestion,
  LiveGenerationPromptContext,
} from './ai/sessions-ai.types';

export type LiveQuestionComposition = {
  mcq: number;
  written: number;
};

/**
 * Gets the target live question composition for a level.
 *
 * @param level The session level.
 * @returns The target three-question composition.
 */
export function getLiveQuestionComposition(
  level: number,
): LiveQuestionComposition {
  if (level >= 31) {
    return { mcq: 0, written: 3 };
  }

  if (level >= 21) {
    return { mcq: 1, written: 2 };
  }

  if (level >= 11) {
    return { mcq: 2, written: 1 };
  }

  return { mcq: 3, written: 0 };
}

/**
 * Gets the next required live question type for live-session progress.
 *
 * @param level The session level.
 * @param acceptedQuestions The currently accepted live-session questions.
 * @returns The next required question type.
 */
export function getNextLiveQuestionType(
  level: number,
  acceptedQuestions: Question[],
): QuestionType {
  const composition = getLiveQuestionComposition(level);
  const acceptedMcq = acceptedQuestions.filter(
    (question) => question.type === QuestionType.MCQ,
  ).length;

  if (acceptedMcq < composition.mcq) {
    return QuestionType.MCQ;
  }

  return QuestionType.Written;
}

/**
 * Creates the prompt for live question generation.
 *
 * @param context The live generation prompt context.
 * @returns The user prompt.
 */
export function createLiveGenerationUserPrompt(
  context: LiveGenerationPromptContext,
): string {
  return JSON.stringify({
    topic: {
      title: context.topicTitle,
      description: context.topicDescription,
      tags: context.topicTags,
    },
    level: context.level,
    questionNumber: context.questionNumber,
    requiredQuestionType: context.questionType,
    acceptedQuestions: context.acceptedQuestions.map((question) => ({
      id: question.id,
      type: question.type,
      prompt: question.prompt,
      targetConcepts: question.targetConcepts,
    })),
    instructions: [
      'Generate exactly one question for the requiredQuestionType.',
      'Do not repeat acceptedQuestions.',
      'Use short option IDs such as o1, o2, and o3 for MCQ questions.',
    ],
  });
}

/**
 * Applies deterministic live question and option IDs.
 *
 * @param question The generated question.
 * @param context The live generation prompt context.
 * @returns The question with formatted IDs.
 */
export function formatGeneratedLiveQuestionIds(
  question: Question,
  context: LiveGenerationPromptContext,
): Question {
  const questionId = createLiveQuestionId(
    context.topicSlug,
    context.level,
    context.questionNumber,
  );

  if (question.type !== QuestionType.MCQ) {
    return { ...question, id: questionId };
  }

  const originalOptions = question.options ?? [];
  const correctOptionIndex = originalOptions.findIndex(
    (option) => option.id === question.correctOptionId,
  );

  if (correctOptionIndex < 0) {
    throw new ServiceUnavailableException(
      'AI response correct option was invalid or not found',
    );
  }

  const options = originalOptions.map((option, index) => ({
    ...option,
    id: createLiveOptionId(questionId, index + 1),
  }));

  return {
    ...question,
    id: questionId,
    options,
    correctOptionId: options[correctOptionIndex].id,
  };
}

/**
 * Parses a generated live question from completion text.
 *
 * @param text The completion text to parse.
 * @returns The parsed generated live question.
 */
export function parseGeneratedLiveQuestion(
  text: string,
): GeneratedLiveQuestion {
  const normalizedText = text
    .trim()
    .replace(/^```json\s*/i, '')
    .replace(/^```\s*/i, '')
    .replace(/```$/i, '')
    .trim();
  let json: unknown;

  try {
    json = JSON.parse(normalizedText);
  } catch {
    throw new ServiceUnavailableException('AI response was invalid');
  }

  const parsed = generatedLiveQuestionSchema.safeParse(json);

  if (!parsed.success) {
    throw new ServiceUnavailableException('AI response was invalid');
  }

  return parsed.data;
}

/**
 * Validates a generated live question before persistence.
 *
 * @param question The generated question.
 * @param expectedType The expected question type.
 */
export function validateGeneratedLiveQuestion(
  question: Question,
  expectedType: QuestionType,
): void {
  if (question.type !== expectedType) {
    throw new ServiceUnavailableException('AI response used wrong type');
  }

  if (question.targetConcepts.length === 0) {
    throw new ServiceUnavailableException('AI response was invalid');
  }

  if (question.type !== QuestionType.MCQ) {
    return;
  }

  const options = question.options ?? [];
  const correctOption = options.find(
    (option) => option.id === question.correctOptionId,
  );

  if (options.length !== 3 || !correctOption) {
    throw new ServiceUnavailableException('AI response was invalid');
  }
}

/**
 * Creates a deterministic live question ID.
 *
 * @param topicSlug The topic slug.
 * @param level The question level.
 * @param questionNumber The question number within the live set.
 * @returns The formatted question ID.
 */
function createLiveQuestionId(
  topicSlug: string,
  level: number,
  questionNumber: number,
): string {
  return `${topicSlug}-l${level}-q${questionNumber}`;
}

/**
 * Creates a deterministic live MCQ option ID.
 *
 * @param questionId The formatted question ID.
 * @param optionNumber The option number within the question.
 * @returns The formatted option ID.
 */
function createLiveOptionId(questionId: string, optionNumber: number): string {
  return `${questionId}-o${optionNumber}`;
}

/**
 * Calculates the average set score from evaluated answers.
 *
 * @param answers The evaluated answers.
 * @returns The average set score.
 */
export function calculateSetScore(answers: Answer[]): number {
  const totalScore = answers.reduce((sum, answer) => sum + answer.score, 0);

  return roundScore(totalScore / answers.length);
}

/**
 * Checks whether every answer reaches the passing threshold.
 *
 * @param answers The evaluated answers.
 * @returns Whether all answers are passing.
 */
export function hasPassingAnswers(answers: Answer[]): boolean {
  return answers.every((answer) => answer.score >= 0.5);
}

/**
 * Calculates the average score from attempts.
 *
 * @param attempts The attempts to score.
 * @returns The average attempt score.
 */
export function calculateAttemptScore(attempts: SetAttemptDocument[]): number {
  if (attempts.length === 0) {
    return 0;
  }

  const totalScore = attempts.reduce(
    (sum, attempt) => sum + attempt.setScore,
    0,
  );

  return roundScore(totalScore / attempts.length);
}

/**
 * Calculates the average score from session evaluations.
 *
 * @param evaluations The session evaluations.
 * @returns The average evaluation score.
 */
export function calculateEvaluationScore(
  evaluations: SessionEvaluationDocument[],
): number {
  const totalScore = evaluations.reduce(
    (sum, evaluation) => sum + evaluation.overallScore,
    0,
  );

  return roundScore(totalScore / evaluations.length);
}

/**
 * Collects unique target concepts matching a score.
 *
 * @param answers The evaluated answers.
 * @param score The answer score to collect concepts for.
 * @returns The matching unique target concepts.
 */
export function collectConceptsByScore(
  answers: Answer[],
  score: number,
): string[] {
  return [
    ...new Set(
      answers
        .filter((answer) => answer.score === score)
        .flatMap((answer) => answer.targetConcepts),
    ),
  ];
}

/**
 * Collects unique attempt concepts from a field.
 *
 * @param attempts The attempts to inspect.
 * @param field The concept field to collect.
 * @returns The unique attempt concepts.
 */
export function collectAttemptConcepts(
  attempts: SetAttemptDocument[],
  field: 'strengths' | 'weaknesses',
): string[] {
  return [...new Set(attempts.flatMap((attempt) => attempt[field] ?? []))];
}

/**
 * Creates recommendations from weaknesses.
 *
 * @param weaknesses The weak concepts to recommend practice for.
 * @returns The generated recommendations.
 */
export function createRecommendations(weaknesses: string[]): string[] {
  return weaknesses.map((concept) => `Review ${concept}.`);
}

/**
 * Rounds a score to one decimal point.
 *
 * @param score The score to round.
 * @returns The rounded score.
 */
export function roundScore(score: number): number {
  return Math.round(score * 10) / 10;
}
