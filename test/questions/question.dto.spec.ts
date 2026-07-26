import { plainToInstance } from 'class-transformer';
import { validate } from 'class-validator';
import { QuestionDto } from '../../src/questions/dtos/question.dto';
import { QuestionType } from '../../src/questions/schemas/question-set.schema';

describe('QuestionDto', () => {
  it('allows written questions to omit MCQ-only fields', async () => {
    const dto = plainToInstance(QuestionDto, {
      id: 'memory-l11-q1',
      type: QuestionType.Written,
      prompt: 'Explain paging in virtual memory.',
      targetConcepts: ['paging'],
      feedback: {
        correct: 'Good explanation.',
        incorrect: 'Review paging.',
      },
      rubrics: {
        keyPoints: ['Pages', 'Frames'],
        misconceptions: ['Paging requires contiguous physical memory'],
      },
    });

    await expect(validate(dto)).resolves.toHaveLength(0);
  });

  it('requires MCQ-only fields for MCQ questions', async () => {
    const dto = plainToInstance(QuestionDto, {
      id: 'memory-l0-q1',
      type: QuestionType.MCQ,
      prompt: 'What does paging manage?',
      targetConcepts: ['paging'],
      feedback: {
        correct: 'Correct.',
        incorrect: 'Review paging.',
      },
      rubrics: {
        keyPoints: ['Pages'],
        misconceptions: ['Segments'],
      },
    });
    const errors = await validate(dto);

    expect(errors.map((error) => error.property)).toEqual(
      expect.arrayContaining(['options', 'correctOptionId']),
    );
  });
});
