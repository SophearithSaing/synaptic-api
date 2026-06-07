import { Body, Controller, Get, Param, Post, UseGuards } from '@nestjs/common';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import { RolesGuard } from '../auth/guards/roles.guard';
import { Roles } from '../auth/decorators/roles.decorator';
import { UserRole } from '../auth/schemas/user.schema';
import { MongoIdPipe } from '../common/pipes/mongo-id.pipe';
import { CategoriesService } from './categories.service';
import { CategoryResponseDto } from './dtos/category-response.dto';
import { CreateCategoryDto } from './dtos/create-category.dto';

/**
 * Controller for managing categories.
 */
@Controller('categories')
@UseGuards(JwtAuthGuard, RolesGuard)
export class CategoriesController {
  /**
   * Creates a categories controller.
   *
   * @param categoriesService The categories service.
   */
  constructor(private categoriesService: CategoriesService) {}

  /**
   * Creates a new topic category.
   *
   * @param body The category details.
   * @returns The created category.
   */
  @Post('category/create')
  @Roles(UserRole.Admin)
  async createCategory(
    @Body() body: CreateCategoryDto,
  ): Promise<CategoryResponseDto> {
    return this.categoriesService.createCategory(body);
  }

  /**
   * Fetches all topic categories.
   *
   * @returns The available topic categories.
   */
  @Get('categories')
  async getCategories(): Promise<CategoryResponseDto[]> {
    return this.categoriesService.getCategories();
  }

  /**
   * Fetches a category by its ID.
   *
   * @param id The category ID.
   * @returns The requested category.
   */
  @Get(':id')
  async getCategoryById(
    @Param('id', MongoIdPipe) id: string,
  ): Promise<CategoryResponseDto> {
    return this.categoriesService.getCategoryById(id);
  }
}
