import {
  Body,
  Controller,
  Delete,
  Get,
  HttpCode,
  HttpStatus,
  Param,
  Post,
  Req,
  UseGuards,
} from '@nestjs/common';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import type { RequestWithUser } from '../auth/types/request-with-user.type';
import { MongoIdPipe } from '../common/pipes/mongo-id.pipe';
import { QuestionSetResponseDto } from '../questions/dtos';
import {
  ContinueLiveSessionDto,
  ContinueSessionDto,
  LiveSessionResponseDto,
  RejectLiveQuestionDto,
  SessionResponseDto,
  StartLiveSessionDto,
  StartLiveSessionResponseDto,
  StartSessionDto,
  StartSessionResponseDto,
  SubmitAnswerDto,
  SubmitAnswerResponseDto,
  SubmitLiveAnswerDto,
  SubmitLiveAnswerResponseDto,
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
   * @returns The created session ID and level 0 question set.
   */
  @Post('start')
  async startSession(
    @Req() request: RequestWithUser,
    @Body() body: StartSessionDto,
  ): Promise<StartSessionResponseDto> {
    return this.sessionsService.startSession(body.topicId, request.user.userId);
  }

  /**
   * Starts a live learning session for the authenticated user.
   *
   * @param request The authenticated request.
   * @param body The live session start request.
   * @returns The created live session and pending generated question.
   */
  @Post('live/start')
  async startLiveSession(
    @Req() request: RequestWithUser,
    @Body() body: StartLiveSessionDto,
  ): Promise<StartLiveSessionResponseDto> {
    return this.sessionsService.startLiveSession(
      body.topicId,
      request.user.userId,
    );
  }

  /**
   * Rejects a live generated question for the authenticated user.
   *
   * @param request The authenticated request.
   * @param body The live question rejection request.
   * @returns The replacement pending generated question.
   */
  @Post('live/reject')
  async rejectLiveQuestion(
    @Req() request: RequestWithUser,
    @Body() body: RejectLiveQuestionDto,
  ): Promise<StartLiveSessionResponseDto> {
    return this.sessionsService.rejectLiveQuestion(
      request.user.userId,
      body.sessionId,
      body.questionId,
      body.reason,
    );
  }

  /**
   * Submits an answer to a live generated question.
   *
   * @param request The authenticated request.
   * @param body The live answer submission request.
   * @returns The evaluated answers and next live question when available.
   */
  @Post('live/submit-answer')
  async submitLiveAnswer(
    @Req() request: RequestWithUser,
    @Body() body: SubmitLiveAnswerDto,
  ): Promise<SubmitLiveAnswerResponseDto> {
    return this.sessionsService.submitLiveAnswer(
      request.user.userId,
      body.sessionId,
      body.questionId,
      body.answer,
    );
  }

  /**
   * Continues a live learning session for the authenticated user.
   *
   * @param request The authenticated request.
   * @param body The live session continue request.
   * @returns The current or next pending generated question.
   */
  @Post('live/continue')
  async continueLiveSession(
    @Req() request: RequestWithUser,
    @Body() body: ContinueLiveSessionDto,
  ): Promise<StartLiveSessionResponseDto> {
    return this.sessionsService.continueLiveSession(
      request.user.userId,
      body.sessionId,
    );
  }

  /**
   * Fetches in-progress sessions for the authenticated user.
   *
   * @param request The authenticated request.
   * @returns The authenticated user's in-progress sessions.
   */
  @Get('in-progress')
  async getInProgressSessions(
    @Req() request: RequestWithUser,
  ): Promise<SessionResponseDto[]> {
    return this.sessionsService.getInProgressSessions(request.user.userId);
  }

  /**
   * Fetches in-progress live sessions for the authenticated user.
   *
   * @param request The authenticated request.
   * @returns The authenticated user's in-progress live sessions.
   */
  @Get('live/in-progress')
  async getInProgressLiveSessions(
    @Req() request: RequestWithUser,
  ): Promise<LiveSessionResponseDto[]> {
    return this.sessionsService.getInProgressLiveSessions(request.user.userId);
  }

  /**
   * Deletes a live learning session for the authenticated user.
   *
   * @param request The authenticated request.
   * @param id The live session ID to delete.
   */
  @Delete('live/:id')
  @HttpCode(HttpStatus.NO_CONTENT)
  async deleteLiveSession(
    @Req() request: RequestWithUser,
    @Param('id', MongoIdPipe) id: string,
  ): Promise<void> {
    await this.sessionsService.deleteLiveSession(id, request.user.userId);
  }

  /**
   * Deletes a learning session for the authenticated user.
   *
   * @param request The authenticated request.
   * @param id The session ID to delete.
   */
  @Delete(':id')
  @HttpCode(HttpStatus.NO_CONTENT)
  async deleteSession(
    @Req() request: RequestWithUser,
    @Param('id', MongoIdPipe) id: string,
  ): Promise<void> {
    await this.sessionsService.deleteSession(id, request.user.userId);
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
