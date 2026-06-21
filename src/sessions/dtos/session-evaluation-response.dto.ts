import { hasToObject } from '../../utils/object.utils';
import { SessionEvaluationDocument } from '../schemas/session-evaluation.schemas';

export class SessionEvaluationResponseDto {
  id: string;
  student: unknown;
  session: unknown;
  topic: unknown;
  fromLevel: number;
  toLevel: number;
  overallScore: number;
  summary: string;
  stength: string[];
  weakness: string[];
  recommendation: string[];
  attemptIds: string[];
  createdAt?: Date;
  updatedAt?: Date;

  /**
   * Creates a response DTO from a session evaluation document.
   *
   * @param evaluation The session evaluation document.
   * @returns The session evaluation response DTO.
   */
  static from(
    evaluation: SessionEvaluationDocument,
  ): SessionEvaluationResponseDto {
    const source = evaluation;

    return {
      id: source._id.toString(),
      student: SessionEvaluationResponseDto.transformReference(source.student),
      session: SessionEvaluationResponseDto.transformReference(source.session),
      topic: SessionEvaluationResponseDto.transformReference(source.topic),
      fromLevel: source.fromLevel,
      toLevel: source.toLevel,
      overallScore: source.overallScore,
      summary: source.summary,
      stength: source.stength,
      weakness: source.weakness,
      recommendation: source.recommendation,
      attemptIds: source.attemptIds,
      createdAt: source.createdAt,
      updatedAt: source.updatedAt,
    };
  }

  /**
   * Creates response DTOs from session evaluation documents.
   *
   * @param evaluations The session evaluation documents.
   * @returns The session evaluation response DTOs.
   */
  static fromMany(
    evaluations: SessionEvaluationDocument[],
  ): SessionEvaluationResponseDto[] {
    return evaluations.map((evaluation) =>
      SessionEvaluationResponseDto.from(evaluation),
    );
  }

  /**
   * Converts a reference into a response value.
   *
   * @param reference The reference or populated document.
   * @returns The response reference value.
   */
  private static transformReference(reference: unknown): unknown {
    if (hasToObject(reference)) {
      return reference.toObject();
    }

    if (reference && typeof reference === 'object') {
      return reference;
    }

    return String(reference);
  }
}
