import { BadRequestException, Injectable, PipeTransform } from '@nestjs/common';
import { Types } from 'mongoose';

/**
 * Validates route parameters that must be MongoDB ObjectIds.
 */
@Injectable()
export class MongoIdPipe implements PipeTransform<string, string> {
  /**
   * Validates and returns a MongoDB ObjectId string.
   *
   * @param value The route parameter value.
   * @returns The validated ObjectId string.
   */
  transform(value: string): string {
    if (!Types.ObjectId.isValid(value)) {
      throw new BadRequestException('Invalid MongoDB ObjectId');
    }

    return value;
  }
}
