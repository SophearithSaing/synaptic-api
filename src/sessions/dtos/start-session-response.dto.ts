import { QuestionSetResponseDto } from '../../questions/dtos';

export class StartSessionResponseDto {
  sessionId: string;
  questionSet: QuestionSetResponseDto;
}
