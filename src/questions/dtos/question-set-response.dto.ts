import { QuestionDto } from './question.dto';
import { QuestionSetDocument } from '../schemas/question-set.schema';

type QuestionSetResponseSource = QuestionSetDocument & {
  _id: { toString(): string };
  createdAt?: Date;
  updatedAt?: Date;
};

export class QuestionSetResponseDto {
  id: string;
  topic: unknown;
  setType: string;
  level: number;
  questions: QuestionDto[];
  createdAt?: Date;
  updatedAt?: Date;

  /**
   * Creates a response DTO from a question set document.
   *
   * @param questionSet The question set document.
   * @returns The question set response DTO.
   */
  static from(questionSet: QuestionSetDocument): QuestionSetResponseDto {
    const source = questionSet as QuestionSetResponseSource;

    return {
      id: source._id.toString(),
      topic: QuestionSetResponseDto.transformTopic(source.topic),
      setType: source.setType,
      level: source.level,
      questions: source.questions,
      createdAt: source.createdAt,
      updatedAt: source.updatedAt,
    };
  }

  /**
   * Creates response DTOs from question set documents.
   *
   * @param questionSets The question set documents.
   * @returns The question set response DTOs.
   */
  static fromMany(
    questionSets: QuestionSetDocument[],
  ): QuestionSetResponseDto[] {
    return questionSets.map((questionSet) =>
      QuestionSetResponseDto.from(questionSet),
    );
  }

  /**
   * Converts a topic reference into a response value.
   *
   * @param topic The topic reference or populated topic.
   * @returns The topic response value.
   */
  private static transformTopic(topic: unknown): unknown {
    if (
      topic &&
      typeof topic === 'object' &&
      'toObject' in topic &&
      typeof topic.toObject === 'function'
    ) {
      return topic.toObject();
    }

    if (topic && typeof topic === 'object') {
      return topic;
    }

    return String(topic);
  }
}
