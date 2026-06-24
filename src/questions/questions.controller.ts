import {
  Body,
  Controller,
  Get,
  Param,
  ParseArrayPipe,
  Patch,
  Post,
  Query,
  UseGuards,
} from '@nestjs/common';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import { RolesGuard } from '../auth/guards/roles.guard';
import { Roles } from '../auth/decorators/roles.decorator';
import { UserRole } from '../auth/schemas/user.schema';
import { MongoIdPipe } from '../common/pipes/mongo-id.pipe';
import {
  CreateQuestionSetDto,
  QuestionSetResponseDto,
  UpdateQuestionSetDto,
} from './dtos';
import { QuestionsService } from './questions.service';

/**
 * Controller for managing question sets.
 */
@Controller('questions')
@UseGuards(JwtAuthGuard, RolesGuard)
export class QuestionsController {
  constructor(private questionsService: QuestionsService) {}

  /**
   * Creates question sets.
   *
   * @param body The question set details.
   * @returns The created question sets.
   */
  @Post('create')
  @Roles(UserRole.Admin)
  async createQuestionSets(
    @Body(new ParseArrayPipe({ items: CreateQuestionSetDto }))
    body: CreateQuestionSetDto[],
  ): Promise<QuestionSetResponseDto[]> {
    return this.questionsService.createQuestionSets(body);
  }

  /**
   * Updates a question set by ID.
   *
   * @param id The question set ID.
   * @param body The question set updates.
   * @returns The updated question set.
   */
  @Patch(':id')
  @Roles(UserRole.Admin)
  async updateQuestionSet(
    @Param('id', MongoIdPipe) id: string,
    @Body() body: UpdateQuestionSetDto,
  ): Promise<QuestionSetResponseDto> {
    return this.questionsService.updateQuestionSet(id, body);
  }

  /**
   * Fetches question sets by topic slug.
   *
   * @param slug The topic slug.
   * @param populateTopic Whether to populate the topic.
   * @returns The question sets for the topic.
   */
  @Get('topic/:slug')
  async getQuestionSetsByTopicSlug(
    @Param('slug') slug: string,
    @Query('populateTopic') populateTopic?: string,
  ): Promise<QuestionSetResponseDto[]> {
    return this.questionsService.getQuestionSetsByTopicSlug(
      slug,
      populateTopic === 'true',
    );
  }

  /**
   * Fetches a question set by ID.
   *
   * @param id The question set ID.
   * @param populateTopic Whether to populate the topic.
   * @returns The requested question set.
   */
  @Get(':id')
  async getQuestionSetById(
    @Param('id', MongoIdPipe) id: string,
    @Query('populateTopic') populateTopic?: string,
  ): Promise<QuestionSetResponseDto> {
    return this.questionsService.getQuestionSetById(
      id,
      populateTopic !== 'false',
    );
  }
}
