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
