import { SessionEvaluationDocument } from './schemas/session-evaluation.schemas';
import { Answer, SetAttemptDocument } from './schemas/set-attempt.schemas';

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
