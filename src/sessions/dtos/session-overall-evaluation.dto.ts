import { ArrayNotEmpty, IsArray, IsNotEmpty, IsString } from 'class-validator';

export class SessionOverallEvaluationDto {
  @IsString()
  @IsNotEmpty()
  summary: string;

  @IsArray()
  @ArrayNotEmpty()
  @IsString({ each: true })
  stengths: string[];

  @IsArray()
  @ArrayNotEmpty()
  @IsString({ each: true })
  weakness: string[];

  @IsArray()
  @ArrayNotEmpty()
  @IsString({ each: true })
  recommendations: string[];
}
