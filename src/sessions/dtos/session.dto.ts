import { Type } from 'class-transformer';
import {
  ArrayNotEmpty,
  IsArray,
  IsEnum,
  IsMongoId,
  IsNotEmpty,
  IsOptional,
  IsString,
  ValidateNested,
} from 'class-validator';
import { AiProvider } from '../../ai/types/ai.types';

/**
 * Request body for starting a learning session.
 */
export class StartSessionDto {
  @IsMongoId()
  @IsNotEmpty()
  topicId: string;

  @IsEnum(AiProvider)
  @IsOptional()
  provider?: AiProvider;
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

  @IsEnum(AiProvider)
  @IsOptional()
  provider?: AiProvider;
}
