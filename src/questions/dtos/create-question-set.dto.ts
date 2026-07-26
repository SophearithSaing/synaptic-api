import { Type } from 'class-transformer';
import {
  ArrayNotEmpty,
  IsArray,
  IsEnum,
  IsMongoId,
  IsNotEmpty,
  IsNumber,
  Min,
  ValidateNested,
} from 'class-validator';
import { QuestionDto } from './question.dto';
import { QuestionSetType } from '../schemas/question-set.schema';

export class CreateQuestionSetDto {
  @IsMongoId()
  @IsNotEmpty()
  topic: string;

  @IsEnum(QuestionSetType)
  @IsNotEmpty()
  setType: QuestionSetType;

  @IsNumber()
  @Min(0)
  level: number;

  @IsArray()
  @ArrayNotEmpty()
  @ValidateNested({ each: true })
  @Type(() => QuestionDto)
  questions: QuestionDto[];
}
