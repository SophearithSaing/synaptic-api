import { Body, Controller, Param, Post, Req, UseGuards } from '@nestjs/common';
import { SessionsService } from './sessions.service';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import type { RequestWithUser } from '../auth/types/request-with-user.type';
import { MongoIdPipe } from '../common/pipes/mongo-id.pipe';
import { StartSessionDto, SubmitSessionDto } from './dto/session.dto';

/**
 * Controller for handling learning session requests.
 */
@Controller('sessions')
@UseGuards(JwtAuthGuard)
export class SessionsController {
  constructor(private readonly sessionsService: SessionsService) {}

  /**
   * Starts a new session for the authenticated user on a given topic.
   *
   * @param req Authenticated request.
   * @param body Session start payload.
   * @returns The newly created session details.
   */
  @Post('start')
  async startSession(
    @Req() req: RequestWithUser,
    @Body() body: StartSessionDto,
  ): Promise<unknown> {
    return this.sessionsService.startSession(
      body.topicId,
      req.user.userId,
      body.provider,
    );
  }

  /**
   * Submits student answers for a specific session for evaluation.
   *
   * @param req Authenticated request.
   * @param id The session QuestionSet ID.
   * @param body Session submission payload.
   * @returns The final evaluation results.
   */
  @Post(':id/submit')
  async submitSession(
    @Req() req: RequestWithUser,
    @Param('id', MongoIdPipe) id: string,
    @Body() body: SubmitSessionDto,
  ): Promise<unknown> {
    return this.sessionsService.submitSession(
      id,
      req.user.userId,
      body.answers,
      body.provider,
    );
  }
}
