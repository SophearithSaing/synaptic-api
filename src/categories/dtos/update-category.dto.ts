import { IsOptional, IsString } from 'class-validator';

export class UpdateCategoryDto {
  @IsString()
  id: string;

  @IsOptional()
  @IsString()
  title?: string;

  @IsOptional()
  @IsString()
  slug?: string;

  @IsOptional()
  @IsString()
  description?: string;

  @IsOptional()
  @IsString()
  icon?: string;
}
