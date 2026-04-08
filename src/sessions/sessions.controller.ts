import { Controller, Post, Body } from '@nestjs/common';
import { SessionsService } from './sessions.service';

/**
 * Controller for handling learning session requests.
 */
@Controller('sessions')
export class SessionsController {
  constructor(private readonly sessionsService: SessionsService) {}

  /**
   * Starts a new session for a user on a given topic.
   * @param topicId Unique identifier for the topic.
   * @param userId Unique identifier for the user.
   * @returns The newly created session details.
   */
  @Post('start')
  async startSession(
    @Body('topicId') topicId: string,
    @Body('userId') userId: string,
  ) {
    return this.sessionsService.startSession(topicId, userId);
  }
}
