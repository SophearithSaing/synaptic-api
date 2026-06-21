import { Type } from 'class-transformer';
import {
  ArrayNotEmpty,
  IsArray,
  IsEnum,
  IsNotEmpty,
  IsOptional,
  IsString,
  ValidateIf,
  ValidateNested,
} from 'class-validator';
import { QuestionType } from '../schemas/question-set.schema';

export class QuestionOptionDto {
  @IsString()
  @IsNotEmpty()
  id: string;

  @IsString()
  @IsNotEmpty()
  text: string;
}

export class QuestionFeedbackDto {
  @IsString()
  @IsNotEmpty()
  correct: string;

  @IsString()
  @IsNotEmpty()
  incorrect: string;
}

export class QuestionRubricDto {
  @IsArray()
  @ArrayNotEmpty()
  @IsString({ each: true })
  keyPoints: string[];

  @IsArray()
  @ArrayNotEmpty()
  @IsString({ each: true })
  misconceptions: string[];
}

export class QuestionDto {
  @IsString()
  @IsNotEmpty()
  id: string;

  @IsEnum(QuestionType)
  @IsNotEmpty()
  type: QuestionType;

  @IsString()
  @IsNotEmpty()
  prompt: string;

  @ValidateIf((obj: QuestionDto) => obj.type === QuestionType.MCQ)
  @IsArray()
  @ValidateNested({ each: true })
  @Type(() => QuestionOptionDto)
  options: QuestionOptionDto[];

  @IsOptional()
  @IsString()
  correctOptionId: string;

  @IsArray()
  @ArrayNotEmpty()
  @IsString({ each: true })
  targetConcepts: string[];

  @ValidateNested()
  @Type(() => QuestionFeedbackDto)
  feedback: QuestionFeedbackDto;

  @ValidateNested()
  @Type(() => QuestionRubricDto)
  rubric: QuestionRubricDto;
}
