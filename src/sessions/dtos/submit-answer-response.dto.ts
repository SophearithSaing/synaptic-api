import { QuestionSetResponseDto } from '../../questions/dtos';
import { SetAttemptResponseDto } from './set-attempt-response.dto';

export class SubmitAnswerResponseDto {
  attempt: SetAttemptResponseDto;
  nextQuestionSet: QuestionSetResponseDto | null;
}
