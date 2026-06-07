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

export class UpdateQuestionSetDto {
  @IsMongoId()
  @IsOptional()
  topic?: string;

  @IsString()
  @IsNotEmpty()
  @IsOptional()
  setType?: string;

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

  @IsNumber()
  @Min(0)
  @IsOptional()
  score?: number;
}
