import { hasToObject } from '../../utils/object.utils';
import { Types } from 'mongoose';
import { LiveSessionDocument } from '../schemas/live-session.schema';
import { SessionStatus } from '../schemas/session.schema';
import { SessionOverallEvaluationDto } from './session-overall-evaluation.dto';

export class LiveSessionResponseDto {
  id: string;
  student: unknown;
  topic: unknown;
  currentLevel: number;
  status: SessionStatus;
  overallEvaluation?: SessionOverallEvaluationDto;
  startedAt?: Date;
  finishedAt?: Date;
  createdAt?: Date;
  updatedAt?: Date;

  /**
   * Creates a response DTO from a live session document.
   *
   * @param session The live session document.
   * @returns The live session response DTO.
   */
  static from(session: LiveSessionDocument): LiveSessionResponseDto {
    const source = session;

    return {
      id: source._id.toString(),
      student: LiveSessionResponseDto.transformReference(source.student),
      topic: LiveSessionResponseDto.transformReference(source.topic),
      currentLevel: source.currentLevel,
      status: source.status,
      overallEvaluation: source.overallEvaluation,
      startedAt: source.startedAt,
      finishedAt: source.finishedAt,
      createdAt: source.createdAt,
      updatedAt: source.updatedAt,
    };
  }

  /**
   * Creates response DTOs from live session documents.
   *
   * @param sessions The live session documents.
   * @returns The live session response DTOs.
   */
  static fromMany(sessions: LiveSessionDocument[]): LiveSessionResponseDto[] {
    return sessions.map((session) => LiveSessionResponseDto.from(session));
  }

  /**
   * Converts a reference into a response value.
   *
   * @param reference The reference or populated document.
   * @returns The response reference value.
   */
  private static transformReference(reference: unknown): unknown {
    if (hasToObject(reference)) {
      return LiveSessionResponseDto.transformDocumentObject(
        reference.toObject(),
      );
    }

    if (LiveSessionResponseDto.isPlainObject(reference)) {
      return LiveSessionResponseDto.transformDocumentObject(reference);
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
        LiveSessionResponseDto.transformDocumentObject(item),
      );
    }

    if (!LiveSessionResponseDto.isPlainObject(value)) {
      return value;
    }

    const source = value as Record<string, unknown>;
    const result: Record<string, unknown> = {};

    if (source['_id']) {
      result['id'] = LiveSessionResponseDto.transformDocumentObjectId(
        source['_id'],
      );
    }

    for (const [key, item] of Object.entries(source)) {
      if (key === '_id') {
        continue;
      }

      result[key] = LiveSessionResponseDto.transformDocumentObject(item);
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

  /**
   * Converts a document object ID into a response ID.
   *
   * @param value The document object ID value.
   * @returns The response ID value.
   */
  private static transformDocumentObjectId(value: unknown): string {
    if (value instanceof Types.ObjectId) {
      return value.toString();
    }

    return String(value);
  }
}
