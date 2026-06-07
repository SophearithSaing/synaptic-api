import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { Category, CategoryDocument } from './schemas/category.schema';
import { CategoryResponseDto } from './dtos/category-response.dto';
import { CreateCategoryDto } from './dtos/create-category.dto';

@Injectable()
export class CategoriesService {
  /**
   * Creates a categories service.
   *
   * @param categoryModel The category model.
   */
  constructor(
    @InjectModel(Category.name)
    private categoryModel: Model<CategoryDocument>,
  ) {}

  /**
   * Creates a new category.
   *
   * @param dto The category details.
   * @returns The created category.
   */
  async createCategory(dto: CreateCategoryDto): Promise<CategoryResponseDto> {
    const category = new this.categoryModel(dto);
    return CategoriesService.toResponse(await category.save());
  }

  /**
   * Fetches all topic categories.
   *
   * @returns The categories sorted by title.
   */
  async getCategories(): Promise<CategoryResponseDto[]> {
    const categories = await this.categoryModel
      .find()
      .sort({ title: 1 })
      .exec();
    return categories.map((category) => CategoriesService.toResponse(category));
  }

  /**
   * Fetches a category by its unique ID.
   *
   * @param id The category ID.
   * @returns The requested category.
   */
  async getCategoryById(id: string): Promise<CategoryResponseDto> {
    const category = await this.categoryModel.findById(id).lean();

    if (!category) {
      throw new NotFoundException('Category not found');
    }

    return CategoriesService.toResponse(category);
  }

  /**
   * Converts a category document to an API response.
   *
   * @param category The category document.
   * @returns The category response.
   */
  static toResponse(category: CategoryDocument): CategoryResponseDto {
    return {
      id: category._id.toString(),
      title: category.title,
      slug: category.slug,
      description: category.description,
      icon: category.icon,
    };
  }
}
