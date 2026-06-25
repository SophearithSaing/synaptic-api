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
      return SessionResponseDto.transformDocumentObject(reference.toObject());
    }

    if (SessionResponseDto.isPlainObject(reference)) {
      return SessionResponseDto.transformDocumentObject(reference);
    }

    return String(reference);
  }

  /**
   * Converts Mongo document objects into API response objects.
   *
   * @param value The object to transform.
   * @returns The transformed response object.
   */
  private static transformDocumentObject(value: unknown): unknown {
    if (Array.isArray(value)) {
      return value.map((item) =>
        SessionResponseDto.transformDocumentObject(item),
      );
    }

    if (!SessionResponseDto.isPlainObject(value)) {
      return value;
    }

    const source = value as Record<string, unknown>;
    const result: Record<string, unknown> = {};

    if (source['_id']) {
      result['id'] = String(source['_id']);
    }

    for (const [key, item] of Object.entries(source)) {
      if (key === '_id') {
        continue;
      }

      result[key] = SessionResponseDto.transformDocumentObject(item);
    }

    return result;
  }

  /**
   * Determines whether a value is a plain object.
   *
   * @param value The value to inspect.
   * @returns Whether the value is a plain object.
   */
  private static isPlainObject(value: unknown): boolean {
    return Object.prototype.toString.call(value) === '[object Object]';
  }
}
