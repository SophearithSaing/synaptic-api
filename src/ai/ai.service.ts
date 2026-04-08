import { Injectable, InternalServerErrorException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import Anthropic from '@anthropic-ai/sdk';
import { GoogleGenAI } from '@google/genai';
import { z } from 'zod';
import { parseAiJson } from './utils/ai.utils';
import { AiProvider, AiModel, ModelExecutionOptions } from './types/ai.types';

/**
 * Service for interacting with AI models (Gemini and Anthropic).
 * Handles question generation and answer evaluation.
 */
@Injectable()
export class AiService {
  private readonly geminiClient: GoogleGenAI;
  private readonly anthropicClient: Anthropic;

  constructor(private readonly configService: ConfigService) {
    const geminiKey = this.configService.get<string>('GEMINI_API_KEY');
    const anthropicKey = this.configService.get<string>('ANTHROPIC_API_KEY');

    this.geminiClient = new GoogleGenAI({ apiKey: geminiKey });
    this.anthropicClient = new Anthropic({
      apiKey: anthropicKey,
    });
  }

  /**
   * Internal mapping to get the correct model for a given provider.
   * @param provider The AI provider.
   * @returns The corresponding AiModel.
   */
  private getModelForProvider(provider: AiProvider): AiModel {
    const modelMap: Record<AiProvider, AiModel> = {
      [AiProvider.Gemini]: AiModel.Gemini_2_5_Flash,
      [AiProvider.Claude]: AiModel.ClaudeSonnet_4_6,
    };
    return modelMap[provider];
  }

  /**
   * Generic method to execute a prompt against the selected AI provider.
   * @param options Configuration for the model call.
   * @returns The raw text response from the model.
   * @throws InternalServerErrorException if the call fails or returns unexpected data.
   */
  private async executePrompt(options: ModelExecutionOptions): Promise<string> {
    try {
      const model = this.getModelForProvider(options.provider);

      if (options.provider === AiProvider.Gemini) {
        const response = await this.geminiClient.models.generateContent({
          model: model,
          contents: options.prompt,
        });
        return response.text ?? '';
      }

      if (options.provider === AiProvider.Claude) {
        const message = await this.anthropicClient.messages.create({
          model: model,
          max_tokens: options.maxTokens || 1024,
          messages: [{ role: 'user', content: options.prompt }],
        });

        const content = message.content[0];
        if (content.type !== 'text') {
          throw new Error('Unexpected response type from Anthropic');
        }
        return content.text;
      }

      throw new Error(`Unsupported model provider: ${options.provider}`);
    } catch (error) {
      throw new InternalServerErrorException(
        `AI request failed (${options.provider}): ${error.message}`,
      );
    }
  }

  /**
   * Generates educational questions.
   * @param topic The subject.
   * @param description Context description.
   * @param difficulty Level 0-100.
   * @param count Number of questions.
   * @param provider Optional AI provider (defaults to Gemini).
   * @returns Array of generated questions.
   */
  async generateQuestions(
    topic: string,
    description: string,
    difficulty: number,
    count: number = 3,
    provider: AiProvider = AiProvider.Gemini,
  ) {
    const prompt = `Generate ${count} educational questions about "${topic}" (Description: ${description}) at difficulty level ${difficulty}.
    The questions should be a mix of MCQ and Written types.
    
    Return the response as a JSON array of objects following this structure:
    [{
      "type": "mcq" | "written",
      "text": "The question text",
      "options": ["Option A", "Option B", "Option C", "Option D"],
      "correctOption": "The correct option",
      "idealAnswerPoints": ["Point 1", "Point 2"],
      "difficulty": ${difficulty}/100
    }]
    
    Ensure the JSON is valid and only return the JSON array.`;

    const responseText = await this.executePrompt({ provider, prompt });
    return parseAiJson(responseText);
  }

  /**
   * Evaluates student's answers.
   * @param answers Array of objects with questionText and studentAnswer.
   * @param provider Optional AI provider (defaults to Gemini).
   * @returns Evaluation result with totalScore, critique, weakConcepts, and strongConcepts.
   */
  async evaluateAnswers(
    answers: { questionText: string; studentAnswer: string }[],
    provider: AiProvider = AiProvider.Gemini,
  ): Promise<{
    totalScore: number;
    critique: string;
    weakConcepts: string[];
    strongConcepts: string[];
  }> {
    const schema = z.object({
      totalScore: z.number().min(0).max(100),
      critique: z.string().max(500),
      weakConcepts: z.array(z.string()),
      strongConcepts: z.array(z.string()),
    });

    const prompt = `Evaluate the following student answers for the corresponding questions:
    
    ${answers
      .map(
        (answer, index) =>
          `Question ${index + 1}: ${answer.questionText}\nStudent Answer ${index + 1}: ${answer.studentAnswer}`,
      )
      .join('\n\n')}
    
    Provide a total score from 0 to 100, a concise critique (max 3 sentences), and identify up to 3 weak concepts and 3 strong concepts based on the student's performance.
    
    Return the response as a JSON object:
    {
      "totalScore": number,
      "critique": "string",
      "weakConcepts": ["concept1", "concept2"],
      "strongConcepts": ["concept3", "concept4"]
    }
    
    Ensure the JSON is valid and only return the JSON object.`;

    const responseText = await this.executePrompt({ provider, prompt });
    const rawResult = parseAiJson(responseText);

    return schema.parse(rawResult);
  }
}
