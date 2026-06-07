import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { Category, CategoryDocument } from './schemas/category.schema';
import { CategoryResponseDto } from './dtos/category-response.dto';
import { CreateCategoryDto } from './dtos/create-category.dto';

@Injectable()
export class CategoriesService {
  constructor(
    @InjectModel(Category.name)
    private categoryModel: Model<CategoryDocument>,
  ) {}

  /**
   * Creates a new category.
   * @param dto The category details.
   * @returns The created category.
   */
  async createCategory(dto: CreateCategoryDto): Promise<CategoryResponseDto> {
    const category = new this.categoryModel(dto);
    return category.save();
  }

  /**
   * Fetches all topic categories.
   * @returns The categories sorted by title.
   */
  async getCategories(): Promise<CategoryResponseDto[]> {
    return this.categoryModel.find().sort({ title: 1 }).exec();
  }

  async getCategoryById(id: string): Promise<CategoryResponseDto> {
    const category = await this.categoryModel.findById(id).lean();

    if (!category) {
      throw new NotFoundException('Category not found');
    }

    return this.toResponse(category);
  }

  private toResponse(category: CategoryDocument): CategoryResponseDto {
    return {
      id: category._id.toString(),
      title: category.title,
      slug: category.slug,
      description: category.description,
      icon: category.icon,
    };
  }
}
