import { Injectable } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model, Types } from 'mongoose';
import { AiLog, AiLogDocument, AiLogOperation } from './schemas/ai-log.schema';
import {
  AiLogPaginatedResponseDto,
  AiLogResponseDto,
} from './dtos/ai-log-response.dto';

@Injectable()
export class AiService {
  constructor(
    @InjectModel(AiLog.name)
    private readonly aiLogModel: Model<AiLogDocument>,
  ) {}

  /**
   * Records a raw question-generation completion.
   *
   * @param prompt The prompt sent to the AI provider.
   * @param output The raw string returned by the AI provider.
   * @returns The created AI log ID.
   */
  async createQuestionGenerationLog(
    prompt: string,
    output: string,
  ): Promise<Types.ObjectId> {
    const aiLog = await this.aiLogModel.create({
      operation: AiLogOperation.QuestionGeneration,
      prompt,
      output,
    });

    return aiLog._id;
  }

  /**
   * Links a recorded completion to its persisted live question.
   *
   * @param aiLogId The AI log ID to update.
   * @param liveQuestionId The generated live question document ID.
   */
  async linkLiveQuestion(
    aiLogId: Types.ObjectId,
    liveQuestionId: Types.ObjectId,
  ): Promise<void> {
    await this.aiLogModel
      .updateOne({ _id: aiLogId }, { $set: { liveQuestion: liveQuestionId } })
      .exec();
  }

  /**
   * Gets a page of AI logs with their linked live questions.
   *
   * @param page The one-based page number.
   * @param limit The maximum number of logs to return.
   * @returns The requested AI log page.
   */
  async getAiLogs(
    page: number,
    limit: number,
  ): Promise<AiLogPaginatedResponseDto> {
    const [aiLogs, total] = await Promise.all([
      this.aiLogModel
        .find()
        .sort({ createdAt: -1 })
        .skip((page - 1) * limit)
        .limit(limit)
        .populate('liveQuestion')
        .exec(),
      this.aiLogModel.countDocuments().exec(),
    ]);

    return {
      items: AiLogResponseDto.fromMany(aiLogs),
      total,
      page,
      limit,
    };
  }
}
