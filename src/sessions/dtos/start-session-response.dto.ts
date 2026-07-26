import { QuestionSetResponseDto } from '../../questions/dtos';
import { QuestionDto } from '../../questions/dtos/question.dto';

export class StartSessionResponseDto {
  sessionId: string;
  questionSet: QuestionSetResponseDto;
}

export class StartLiveSessionResponseDto {
  sessionId: string;
  questionId: string;
  question: QuestionDto;
}
