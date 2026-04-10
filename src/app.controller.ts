import { Controller, Get } from '@nestjs/common';
import { AppService } from './app.service';

/**
 * Root controller for the application, providing basic health checks or welcome messages.
 */
@Controller()
export class AppController {
  constructor(private readonly appService: AppService) {}

  /**
   * Returns a basic greeting.
   * @returns Greeting string.
   */
  @Get()
  getHello(): string {
    return this.appService.getHello();
  }
}
