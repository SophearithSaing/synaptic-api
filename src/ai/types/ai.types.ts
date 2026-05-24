/**
 * Supported AI providers.
 */
export enum AiProvider {
  Gemini = 'gemini',
  Claude = 'claude',
}

/**
 * Supported AI models (Internal mapping).
 */
export enum AiModel {
  Gemini_3_Flash = 'gemini-3-flash-preview',
  Claude_4_6_Opus = 'claude-opus-4-6',
}

/**
 * Configuration for executing a model call.
 */
export interface ModelExecutionOptions {
  provider: AiProvider;
  prompt: string;
  maxTokens?: number;
}

/**
 * Student answer payload sent to the evaluator model.
 */
export interface AiAnswerEvaluationInput {
  questionId: string;
  questionText: string;
  studentAnswer: string;
}

/**
 * Evaluation details for a single question.
 */
export interface AiQuestionEvaluation {
  questionId: string;
  score: number;
  isCorrect: boolean;
  feedback: string;
}

/**
 * Structured answer evaluation returned by the evaluator model.
 */
export interface AiAnswerEvaluation {
  totalScore: number;
  critique: string;
  weakConcepts: string[];
  strongConcepts: string[];
  questionEvaluations: AiQuestionEvaluation[];
}
