import { Type } from 'class-transformer';
import {
  ArrayNotEmpty,
  IsArray,
  IsEnum,
  IsMongoId,
  IsNotEmpty,
  IsNumber,
  IsOptional,
  Min,
  ValidateNested,
} from 'class-validator';
import { QuestionDto } from './question.dto';
import { QuestionSetType } from '../schemas/question-set.schema';

export class UpdateQuestionSetDto {
  @IsMongoId()
  @IsOptional()
  topic?: string;

  @IsEnum(QuestionSetType)
  @IsNotEmpty()
  @IsOptional()
  setType?: QuestionSetType;

  @IsNumber()
  @Min(0)
  @IsOptional()
  level?: number;

  @IsArray()
  @ArrayNotEmpty()
  @ValidateNested({ each: true })
  @Type(() => QuestionDto)
  @IsOptional()
  questions?: QuestionDto[];
}
