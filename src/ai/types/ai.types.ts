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
  Gemini_1_5_Flash = 'gemini-1.5-flash',
  Claude_3_5_Sonnet = 'claude-3-5-sonnet-20241022',
}

/**
 * Configuration for executing a model call.
 */
export interface ModelExecutionOptions {
  provider: AiProvider;
  prompt: string;
  maxTokens?: number;
}
