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
  @IsNotEmpty()
  correctAnswer: string;

  @IsNumber()
  score: number;

  @IsString()
  feedback: string;

  @IsArray()
  @IsString({ each: true })
  targetConcepts: string[];

  @IsArray()
  @IsString({ each: true })
  strengths: string[];

  @IsArray()
  @IsString({ each: true })
  weaknesses: string[];

  @IsEnum(EvaluatedBy)
  evaluatedBy: EvaluatedBy;
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
  level: number;

  @IsArray()
  @ArrayNotEmpty()
  @ValidateNested({ each: true })
  @Type(() => SetAttemptAnswerDto)
  answers: SetAttemptAnswerDto[];

  @IsNumber()
  setScore: number;

  @IsBoolean()
  passed: boolean;

  @IsArray()
  @IsString({ each: true })
  strengths: string[];

  @IsArray()
  @IsString({ each: true })
  weaknesses: string[];

  @IsString()
  @IsOptional()
  aiSummary?: string;

  @IsDate()
  @Type(() => Date)
  submittedAt: Date;

  @IsDate()
  @Type(() => Date)
  evaluatedAt: Date;
}
