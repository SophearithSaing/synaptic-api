import { Types } from 'mongoose';
import { QuestionDto } from '../../questions/dtos/question.dto';
import { LiveQuestionDocument } from '../../sessions/schemas/live-question.schema';
import { AiLogDocument, AiLogOperation } from '../schemas/ai-log.schema';

type AiLogResponseSource = Omit<AiLogDocument, 'liveQuestion'> & {
  liveQuestion?: LiveQuestionDocument | Types.ObjectId;
};

export class AiLogLiveQuestionResponseDto {
  id: string;
  question: QuestionDto;
  level: number;
  questionNumber: number;
  status: string;

  /**
   * Creates a live question response from a persisted live question.
   *
   * @param liveQuestion The populated live question.
   * @returns The live question response.
   */
  static from(
    liveQuestion: LiveQuestionDocument,
  ): AiLogLiveQuestionResponseDto {
    return {
      id: liveQuestion._id.toString(),
      question: liveQuestion.question,
      level: liveQuestion.level,
      questionNumber: liveQuestion.questionNumber,
      status: liveQuestion.status,
    };
  }
}

export class AiLogResponseDto {
  id: string;
  operation: AiLogOperation;
  prompt: string;
  output: string;
  liveQuestion: AiLogLiveQuestionResponseDto | null;
  createdAt: Date;

  /**
   * Creates an AI log response from a persisted AI log.
   *
   * @param aiLog The AI log document.
   * @returns The AI log response.
   */
  static from(aiLog: AiLogDocument): AiLogResponseDto {
    const source = aiLog as AiLogResponseSource;
    const liveQuestion = source.liveQuestion;

    return {
      id: source._id.toString(),
      operation: source.operation,
      prompt: source.prompt,
      output: source.output,
      liveQuestion:
        liveQuestion && !(liveQuestion instanceof Types.ObjectId)
          ? AiLogLiveQuestionResponseDto.from(liveQuestion)
          : null,
      createdAt: source.createdAt,
    };
  }

  /**
   * Creates AI log responses from persisted AI logs.
   *
   * @param aiLogs The AI log documents.
   * @returns The AI log responses.
   */
  static fromMany(aiLogs: AiLogDocument[]): AiLogResponseDto[] {
    return aiLogs.map((aiLog) => AiLogResponseDto.from(aiLog));
  }
}

export class AiLogPaginatedResponseDto {
  items: AiLogResponseDto[];
  total: number;
  page: number;
  limit: number;
}
