import { Controller, Get, Query, UseGuards } from '@nestjs/common';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import { Roles } from '../auth/decorators/roles.decorator';
import { RolesGuard } from '../auth/guards/roles.guard';
import { UserRole } from '../auth/schemas/user.schema';
import { GetAiLogsQueryDto } from './dtos/get-ai-logs-query.dto';
import { AiLogPaginatedResponseDto } from './dtos/ai-log-response.dto';
import { AiService } from './ai.service';

@Controller('ai/logs')
@UseGuards(JwtAuthGuard, RolesGuard)
@Roles(UserRole.Admin)
export class AiController {
  constructor(private readonly aiService: AiService) {}

  /**
   * Gets paginated AI completion logs.
   *
   * @param query The pagination options.
   * @returns The requested AI log page.
   */
  @Get()
  async getAiLogs(
    @Query() query: GetAiLogsQueryDto,
  ): Promise<AiLogPaginatedResponseDto> {
    return this.aiService.getAiLogs(query.page, query.limit);
  }
}
