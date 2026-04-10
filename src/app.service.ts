import { Injectable } from '@nestjs/common';

/**
 * Basic application service for root-level operations.
 */
@Injectable()
export class AppService {
  /**
   * Returns a standard greeting message.
   * @returns 'Hello World!' string.
   */
  getHello(): string {
    return 'Hello World!';
  }
}
