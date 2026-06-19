import { Body, Controller, Post, Req, UseGuards } from '@nestjs/common';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import type { RequestWithUser } from '../auth/types/request-with-user.type';
import { SessionResponseDto, StartSessionDto } from './dtos';
import { SessionsService } from './sessions.service';

@Controller('sessions')
@UseGuards(JwtAuthGuard)
export class SessionsController {
  constructor(private readonly sessionsService: SessionsService) {}

  /**
   * Starts a learning session for the authenticated user.
   *
   * @param request The authenticated request.
   * @param body The session start request.
   * @returns The created session.
   */
  @Post('start')
  async startSession(
    @Req() request: RequestWithUser,
    @Body() body: StartSessionDto,
  ): Promise<SessionResponseDto> {
    return this.sessionsService.startSession(body.topicId, request.user.userId);
  }
}
