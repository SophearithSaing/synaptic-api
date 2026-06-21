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
