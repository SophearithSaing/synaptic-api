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
  Gemini_2_5_Flash = 'gemini-2.5-flash',
  ClaudeSonnet_4_6 = 'claude-sonnet-4-6',
}

/**
 * Configuration for executing a model call.
 */
export interface ModelExecutionOptions {
  provider: AiProvider;
  prompt: string;
  maxTokens?: number;
}
