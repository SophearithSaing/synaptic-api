import { hasToObject } from '../../utils/object.utils';
import { SessionDocument } from '../schemas/session.schema';
import { SessionOverallEvaluationDto } from './session-overall-evaluation.dto';

export class SessionResponseDto {
  id: string;
  student: unknown;
  topic: unknown;
  currentLevel: number;
  status: string;
  overallEvaluation?: SessionOverallEvaluationDto;
  startAt?: Date;
  finishAt?: Date;
  createdAt?: Date;
  updatedAt?: Date;

  /**
   * Creates a response DTO from a session document.
   *
   * @param session The session document.
   * @returns The session response DTO.
   */
  static from(session: SessionDocument): SessionResponseDto {
    const source = session;

    return {
      id: source._id.toString(),
      student: SessionResponseDto.transformReference(source.student),
      topic: SessionResponseDto.transformReference(source.topic),
      currentLevel: source.currentLevel,
      status: source.status,
      overallEvaluation: source.overallEvaluation,
      startAt: source.startAt,
      finishAt: source.finishAt,
      createdAt: source.createdAt,
      updatedAt: source.updatedAt,
    };
  }

  /**
   * Creates response DTOs from session documents.
   *
   * @param sessions The session documents.
   * @returns The session response DTOs.
   */
  static fromMany(sessions: SessionDocument[]): SessionResponseDto[] {
    return sessions.map((session) => SessionResponseDto.from(session));
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
