import { Controller, Post, Body, Param } from '@nestjs/common';
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

  /**
   * Submits student answers for a specific session for evaluation.
   * @param id The session (QuestionSet) ID.
   * @param answers Array of student answers with their corresponding question IDs.
   * @returns The final evaluation results and updated student profile.
   */
  @Post(':id/submit')
  async submitSession(
    @Param('id') id: string,
    @Body('answers') answers: { questionId: string; studentAnswer: string }[],
  ) {
    return this.sessionsService.submitSession(id, answers);
  }
}
