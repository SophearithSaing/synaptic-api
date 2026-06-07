import { IsString } from 'class-validator';

export class CategoryResponseDto {
  @IsString()
  id: string;

  @IsString()
  title: string;

  @IsString()
  slug: string;

  @IsString()
  description: string;

  @IsString()
  icon: string;
}
