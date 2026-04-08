import { Injectable } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import Anthropic from '@anthropic-ai/sdk';
import { GoogleGenAI } from '@google/genai';
import { z } from 'zod';

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
   * Returns the Gemini AI client.
   * @returns GoogleGenAI instance.
   */
  getGeminiClient() {
    return this.geminiClient;
  }

  /**
   * Returns the Anthropic AI client.
   * @returns Anthropic instance.
   */
  getAnthropicClient() {
    return this.anthropicClient;
  }

  /**
   * Generates a set of educational questions using Gemini 1.5 Flash.
   * @param topic The subject of the questions.
   * @param description A brief description of the topic context.
   * @param difficulty Difficulty level (0-100).
   * @param count Number of questions to generate.
   * @returns Array of generated question objects.
   */
  async generateQuestions(
    topic: string,
    description: string,
    difficulty: number,
    count: number = 3,
  ) {
    const prompt = `Generate ${count} educational questions about "${topic}" (Description: ${description}) at difficulty level ${difficulty}.
    The questions should be a mix of MCQ and Written types.
    
    Return the response as a JSON array of objects following this structure:
    {
      "type": "mcq" | "written",
      "text": "The question text",
      "options": ["Option A", "Option B", "Option C", "Option D"], // Only for MCQ
      "correctOption": "The correct option", // Only for MCQ
      "idealAnswerPoints": ["Point 1", "Point 2"], // Only for Written
      "difficulty": ${difficulty}/100
    }
    
    Ensure the JSON is valid and only return the JSON array.`;

    const response = await this.geminiClient.models.generateContent({
      model: 'gemini-1.5-flash',
      contents: prompt,
    });
    const responseText = response.text ?? '';

    const jsonStr = responseText.replace(/```json|```/g, '').trim();
    return JSON.parse(jsonStr);
  }

  /**
   * Evaluates student's answers using Claude 3.5.
   * @param answers Array of objects with questionText and studentAnswer.
   * @returns Evaluation result with totalScore and critique.
   */
  async evaluateAnswers(
    answers: { questionText: string; studentAnswer: string }[],
  ): Promise<{ totalScore: number; critique: string }> {
    const schema = z.object({
      totalScore: z.number().min(0).max(100),
      critique: z.string().max(500),
    });

    const prompt = `Evaluate the following student answers for the corresponding questions:
    
    ${answers
      .map(
        (a, i) =>
          `Question ${i + 1}: ${a.questionText}\nStudent Answer ${i + 1}: ${a.studentAnswer}`,
      )
      .join('\n\n')}
    
    Provide a total score from 0 to 100 based on the accuracy and completeness of the answers.
    Also, provide a concise critique of the student's performance (maximum 3 sentences).
    
    Return the response as a JSON object with the following structure:
    {
      "totalScore": number,
      "critique": "string"
    }
    
    Ensure the JSON is valid and only return the JSON object.`;

    const message = await this.anthropicClient.messages.create({
      model: 'claude-3-5-sonnet-20241022',
      max_tokens: 1024,
      messages: [{ role: 'user', content: prompt }],
    });

    const content = message.content[0];
    if (content.type !== 'text') {
      throw new Error('Unexpected response type from Anthropic');
    }

    const responseText = content.text;
    const jsonStr = responseText.replace(/```json|```/g, '').trim();
    const rawResult = JSON.parse(jsonStr);

    return schema.parse(rawResult);
  }
}
