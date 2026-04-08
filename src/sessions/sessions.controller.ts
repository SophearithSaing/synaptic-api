import { Controller, Post, Body } from '@nestjs/common';
import { SessionsService } from './sessions.service';

@Controller('sessions')
export class SessionsController {
  constructor(private readonly sessionsService: SessionsService) {}

  @Post('start')
  async startSession(
    @Body('topicId') topicId: string,
    @Body('userId') userId: string,
  ) {
    return this.sessionsService.startSession(topicId, userId);
  }
}
