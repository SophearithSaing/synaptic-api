import { Type } from 'class-transformer';
import {
  ArrayNotEmpty,
  IsArray,
  IsNotEmpty,
  IsString,
  ValidateNested,
} from 'class-validator';

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

  @IsString()
  @IsNotEmpty()
  type: string;

  @IsString()
  @IsNotEmpty()
  prompt: string;

  @IsArray()
  @ValidateNested({ each: true })
  @Type(() => QuestionOptionDto)
  options: QuestionOptionDto[];

  @IsString()
  @IsNotEmpty()
  correctionOptionId: string;

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
