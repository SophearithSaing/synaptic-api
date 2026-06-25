import { SessionResponseDto } from '../../src/sessions/dtos/session-response.dto';

describe('SessionResponseDto', () => {
  it('converts populated topic _id to id', () => {
    const session = {
      _id: { toString: () => 'session-id' },
      student: { toString: () => 'student-id' },
      topic: {
        toObject: () => ({
          _id: { toString: () => 'topic-id' },
          title: 'Memory Management',
          slug: 'memory-management',
          tags: ['systems'],
        }),
      },
      currentLevel: 1,
      status: 'active',
      startAt: new Date('2026-01-01T00:00:00.000Z'),
      finishAt: undefined,
      createdAt: new Date('2026-01-01T00:00:00.000Z'),
      updatedAt: new Date('2026-01-01T00:00:00.000Z'),
    } as never;

    expect(SessionResponseDto.from(session).topic).toEqual({
      id: 'topic-id',
      title: 'Memory Management',
      slug: 'memory-management',
      tags: ['systems'],
    });
  });
});
