import { Body, Controller, Post, Req, UseGuards } from '@nestjs/common';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import type { RequestWithUser } from '../auth/types/request-with-user.type';
import { QuestionSetResponseDto } from '../questions/dtos';
import {
  ContinueSessionDto,
  StartSessionDto,
  SubmitAnswerDto,
  SubmitAnswerResponseDto,
} from './dtos';
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
   * @returns The level 0 question set for the selected topic.
   */
  @Post('start')
  async startSession(
    @Req() request: RequestWithUser,
    @Body() body: StartSessionDto,
  ): Promise<QuestionSetResponseDto> {
    return this.sessionsService.startSession(body.topicId, request.user.userId);
  }

  /**
   * Continues a learning session for the authenticated user.
   *
   * @param request The authenticated request.
   * @param body The session continue request.
   * @returns The question set for the session's current level.
   */
  @Post('continue')
  async continueSession(
    @Req() request: RequestWithUser,
    @Body() body: ContinueSessionDto,
  ): Promise<QuestionSetResponseDto> {
    return this.sessionsService.continueSession(
      body.sessionId,
      request.user.userId,
    );
  }

  /**
   * Submits answers for the authenticated user's current question set.
   *
   * @param request The authenticated request.
   * @param body The submitted answers request.
   * @returns The created set attempt and next question set when available.
   */
  @Post('submit-answer')
  async submitAnswer(
    @Req() request: RequestWithUser,
    @Body() body: SubmitAnswerDto,
  ): Promise<SubmitAnswerResponseDto> {
    return this.sessionsService.submitAnswer(
      request.user.userId,
      body.sessionId,
      body.questionSetId,
      body.answers,
    );
  }
}
