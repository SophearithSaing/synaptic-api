import { IsArray, IsObject, IsString } from 'class-validator';
import { CategoryResponseDto } from '../../categories/dtos/category-response.dto';
import { Type } from 'class-transformer';

export class TopicResponseDto {
  @IsString()
  id: string;

  @IsString()
  title: string;

  @IsObject()
  @Type(() => CategoryResponseDto)
  category: CategoryResponseDto;

  @IsString()
  slug: string;

  @IsString()
  description: string;

  @IsString()
  icon: string;

  @IsArray({ each: true })
  @IsString()
  tags: string[];
}
