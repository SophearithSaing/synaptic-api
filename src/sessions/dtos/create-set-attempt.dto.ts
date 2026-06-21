import { Type } from 'class-transformer';
import {
  ArrayNotEmpty,
  IsArray,
  IsBoolean,
  IsDate,
  IsEnum,
  IsMongoId,
  IsNotEmpty,
  IsNumber,
  IsOptional,
  IsString,
  ValidateNested,
} from 'class-validator';
import { QuestionType } from '../../questions/schemas/question-set.schema';
import { EvaluatedBy } from '../schemas/set-attempt.schemas';

export class SetAttemptAnswerDto {
  @IsString()
  @IsNotEmpty()
  id: string;

  @IsString()
  @IsNotEmpty()
  questionId: string;

  @IsEnum(QuestionType)
  @IsNotEmpty()
  questionType: QuestionType;

  @IsString()
  @IsNotEmpty()
  answer: string;

  @IsString()
  @IsOptional()
  correctAnswer?: string;

  @IsNumber()
  @IsOptional()
  score?: number;

  @IsString()
  @IsOptional()
  feedback?: string;

  @IsArray()
  @IsString({ each: true })
  @IsOptional()
  targetConcepts?: string[];

  @IsArray()
  @IsString({ each: true })
  @IsOptional()
  strength?: string[];

  @IsArray()
  @IsString({ each: true })
  @IsOptional()
  weakness?: string[];

  @IsEnum(EvaluatedBy)
  @IsOptional()
  evaluatedBy?: EvaluatedBy;
}

export class CreateSetAttemptDto {
  @IsMongoId()
  @IsNotEmpty()
  user: string;

  @IsMongoId()
  @IsNotEmpty()
  session: string;

  @IsMongoId()
  @IsNotEmpty()
  topic: string;

  @IsMongoId()
  @IsNotEmpty()
  questionSet: string;

  @IsNumber()
  @IsOptional()
  level?: number;

  @IsArray()
  @ArrayNotEmpty()
  @ValidateNested({ each: true })
  @Type(() => SetAttemptAnswerDto)
  @IsOptional()
  answers?: SetAttemptAnswerDto[];

  @IsNumber()
  @IsOptional()
  setScore?: number;

  @IsBoolean()
  @IsOptional()
  passed?: boolean;

  @IsArray()
  @IsString({ each: true })
  @IsOptional()
  strength?: string[];

  @IsArray()
  @IsString({ each: true })
  @IsOptional()
  weakness?: string[];

  @IsString()
  @IsOptional()
  aiSummary?: string;

  @IsDate()
  @Type(() => Date)
  @IsOptional()
  submittedAt?: Date;

  @IsDate()
  @Type(() => Date)
  @IsOptional()
  evaluatedAt?: Date;
}
