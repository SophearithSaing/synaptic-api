import { Type } from 'class-transformer';
import {
  ArrayNotEmpty,
  IsArray,
  IsMongoId,
  IsNotEmpty,
  IsNumber,
  IsOptional,
  IsString,
  Min,
  ValidateNested,
} from 'class-validator';
import { QuestionDto } from './question.dto';

export class CreateQuestionSetDto {
  @IsMongoId()
  @IsNotEmpty()
  topic: string;

  @IsString()
  @IsNotEmpty()
  setType: string;

  @IsNumber()
  @Min(0)
  level: number;

  @IsArray()
  @ArrayNotEmpty()
  @ValidateNested({ each: true })
  @Type(() => QuestionDto)
  questions: QuestionDto[];

  @IsNumber()
  @Min(0)
  @IsOptional()
  score?: number;
}
