import { Injectable } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import Anthropic from '@anthropic-ai/sdk';
import { GoogleGenAI } from '@google/genai';

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

  getGeminiClient() {
    return this.geminiClient;
  }

  getAnthropicClient() {
    return this.anthropicClient;
  }

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
      model: 'gemini-2.5-flash',
      contents: prompt,
    });
    const responseText = response.text ?? '';

    const jsonStr = responseText.replace(/```json|```/g, '').trim();
    return JSON.parse(jsonStr);
  }
}
