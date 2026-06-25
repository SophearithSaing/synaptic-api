import { Answer, SetAttemptDocument } from '../schemas/set-attempt.schemas';

export class SetAttemptResponseDto {
  id: string;
  user: string;
  session: string;
  topic: string;
  questionSet: string;
  level?: number;
  answers?: Answer[];
  setScore?: number;
  passed?: boolean;
  strengths?: string[];
  weaknesses?: string[];
  aiSummary?: string;
  submittedAt?: Date;
  evaluatedAt?: Date;
  createdAt?: Date;
  updatedAt?: Date;

  /**
   * Creates a response DTO from a set attempt document.
   *
   * @param attempt The set attempt document.
   * @returns The set attempt response DTO.
   */
  static from(attempt: SetAttemptDocument): SetAttemptResponseDto {
    const source = attempt;

    return {
      id: source._id.toString(),
      user: source.user.toString(),
      session: source.session.toString(),
      topic: source.topic.toString(),
      questionSet: source.questionSet.toString(),
      level: source.level,
      answers: source.answers,
      setScore: source.setScore,
      passed: source.passed,
      strengths: source.strengths,
      weaknesses: source.weaknesses,
      aiSummary: source.aiSummary,
      submittedAt: source.submittedAt,
      evaluatedAt: source.evaluatedAt,
      createdAt: source.createdAt,
      updatedAt: source.updatedAt,
    };
  }

  /**
   * Creates response DTOs from set attempt documents.
   *
   * @param attempts The set attempt documents.
   * @returns The set attempt response DTOs.
   */
  static fromMany(attempts: SetAttemptDocument[]): SetAttemptResponseDto[] {
    return attempts.map((attempt) => SetAttemptResponseDto.from(attempt));
  }
}
