import { BadRequestException } from '@nestjs/common';
import { MongoIdPipe } from '../../../src/common/pipes/mongo-id.pipe';

describe('MongoIdPipe', () => {
  let pipe: MongoIdPipe;

  beforeEach(() => {
    pipe = new MongoIdPipe();
  });

  it('returns valid MongoDB ObjectIds', () => {
    const id = '507f1f77bcf86cd799439011';

    expect(pipe.transform(id)).toBe(id);
  });

  it('rejects invalid MongoDB ObjectIds', () => {
    expect(() => pipe.transform('invalid')).toThrow(BadRequestException);
  });
});
