import { QuestionSetResponseDto } from '../../questions/dtos';
import { StartLiveSessionResponseDto } from './start-session-response.dto';
import { Answer } from '../schemas/set-attempt.schemas';
import { SetAttemptResponseDto } from './set-attempt-response.dto';

export class SubmitAnswerResponseDto {
  attempt: SetAttemptResponseDto;
  nextQuestionSet: QuestionSetResponseDto | null;
}

export class SubmitLiveAnswerResponseDto {
  answers: Answer[];
  nextQuestion: StartLiveSessionResponseDto | null;
}
