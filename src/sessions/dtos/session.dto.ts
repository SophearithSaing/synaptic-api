import { Type } from 'class-transformer';
import {
  ArrayNotEmpty,
  IsArray,
  IsMongoId,
  IsNotEmpty,
  IsString,
  ValidateNested,
} from 'class-validator';

/**
 * Request body for starting a learning session.
 */
export class StartSessionDto {
  @IsMongoId()
  @IsNotEmpty()
  topicId: string;
}

/**
 * Request body for continuing a learning session.
 */
export class ContinueSessionDto {
  @IsMongoId()
  @IsNotEmpty()
  sessionId: string;
}

/**
 * Submitted answer for a question set question.
 */
export class SubmitAnswerItemDto {
  @IsString()
  @IsNotEmpty()
  questionId: string;

  @IsString()
  @IsNotEmpty()
  answer: string;
}

/**
 * Request body for submitting answers for a question set.
 */
export class SubmitAnswerDto {
  @IsMongoId()
  @IsNotEmpty()
  sessionId: string;

  @IsMongoId()
  @IsNotEmpty()
  questionSetId: string;

  @IsArray()
  @ArrayNotEmpty()
  @ValidateNested({ each: true })
  @Type(() => SubmitAnswerItemDto)
  answers: SubmitAnswerItemDto[];
}

/**
 * Student answer submitted for a generated question.
 */
export class SessionAnswerDto {
  @IsMongoId()
  @IsNotEmpty()
  questionId: string;

  @IsString()
  @IsNotEmpty()
  studentAnswer: string;
}

/**
 * Request body for submitting a learning session.
 */
export class SubmitSessionDto {
  @IsArray()
  @ArrayNotEmpty()
  @ValidateNested({ each: true })
  @Type(() => SessionAnswerDto)
  answers: SessionAnswerDto[];
}
