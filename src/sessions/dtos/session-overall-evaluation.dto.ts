import { ArrayNotEmpty, IsArray, IsNotEmpty, IsString } from 'class-validator';

export class SessionOverallEvaluationDto {
  @IsString()
  @IsNotEmpty()
  summary: string;

  @IsArray()
  @ArrayNotEmpty()
  @IsString({ each: true })
  strengths: string[];

  @IsArray()
  @ArrayNotEmpty()
  @IsString({ each: true })
  weaknesses: string[];

  @IsArray()
  @ArrayNotEmpty()
  @IsString({ each: true })
  recommendations: string[];
}
