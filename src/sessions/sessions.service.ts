import {
  BadRequestException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model, Types } from 'mongoose';
import { QuestionSetResponseDto } from '../questions/dtos';
import {
  Question,
  QuestionSet,
  QuestionSetDocument,
  QuestionType,
} from '../questions/schemas/question-set.schema';
import { Topic, TopicDocument } from '../topics/schemas/topic.schema';
import { SetAttemptResponseDto, SubmitAnswerItemDto } from './dtos';
import { Session, SessionDocument } from './schemas/session.schema';
import {
  Answer,
  EvaluatedBy,
  SetAttempt,
  SetAttemptDocument,
} from './schemas/set-attempt.schemas';

@Injectable()
export class SessionsService {
  constructor(
    @InjectModel(Session.name)
    private readonly sessionModel: Model<SessionDocument>,
    @InjectModel(Topic.name)
    private readonly topicModel: Model<TopicDocument>,
    @InjectModel(QuestionSet.name)
    private readonly questionSetModel: Model<QuestionSetDocument>,
    @InjectModel(SetAttempt.name)
    private readonly setAttemptModel: Model<SetAttemptDocument>,
  ) {}

  /**
   * Starts a learning session for a user on a topic.
   *
   * @param topicId The topic ID to start.
   * @param studentId The authenticated student ID.
   * @returns The level 0 question set for the selected topic.
   */
  async startSession(
    topicId: string,
    studentId: string,
  ): Promise<QuestionSetResponseDto> {
    const topic = await this.topicModel.findById(topicId).exec();

    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    const questionSet = await this.questionSetModel
      .findOne({ topic: topic._id.toString(), level: 0 })
      .exec();

    if (!questionSet) {
      throw new NotFoundException('Question set not found');
    }

    await this.sessionModel.create({
      student: Types.ObjectId.createFromHexString(studentId),
      topic: topic._id,
      currentLevel: 0,
      status: 'active',
      startAt: new Date(),
    });

    return QuestionSetResponseDto.from(questionSet);
  }

  /**
   * Continues a learning session for a user.
   *
   * @param sessionId The session ID to continue.
   * @param studentId The authenticated student ID.
   * @returns The question set for the session's current level.
   */
  async continueSession(
    sessionId: string,
    studentId: string,
  ): Promise<QuestionSetResponseDto> {
    const session = await this.sessionModel
      .findOne({
        _id: Types.ObjectId.createFromHexString(sessionId),
        student: Types.ObjectId.createFromHexString(studentId),
      })
      .exec();

    if (!session) {
      throw new NotFoundException('Session not found');
    }

    const questionSet = await this.questionSetModel
      .findOne({
        topic: session.topic.toString(),
        level: session.currentLevel,
      })
      .exec();

    if (!questionSet) {
      throw new NotFoundException('Question set not found');
    }

    return QuestionSetResponseDto.from(questionSet);
  }

  /**
   * Submits answers for the current session question set.
   *
   * @param studentId The authenticated student ID.
   * @param sessionId The session ID receiving submitted answers.
   * @param questionSetId The question set ID being submitted.
   * @param submittedAnswers The submitted answers.
   * @returns The created set attempt.
   */
  async submitAnswer(
    studentId: string,
    sessionId: string,
    questionSetId: string,
    submittedAnswers: SubmitAnswerItemDto[],
  ): Promise<SetAttemptResponseDto> {
    const session = await this.sessionModel
      .findOne({
        _id: Types.ObjectId.createFromHexString(sessionId),
        student: Types.ObjectId.createFromHexString(studentId),
      })
      .exec();

    if (!session) {
      throw new NotFoundException('Session not found');
    }

    const questionSet = await this.questionSetModel
      .findById(questionSetId)
      .exec();

    if (!questionSet) {
      throw new NotFoundException('Question set not found');
    }

    const answers = submittedAnswers.map((submittedAnswer) =>
      this.evaluateAnswer(submittedAnswer, questionSet.questions),
    );
    const setScore = this.calculateSetScore(answers);
    const passed = setScore >= 0.7;
    const strength = this.collectConceptsByScore(answers, 1);
    const weakness = this.collectConceptsByScore(answers, 0);
    const submittedAt = new Date();
    const attempt = await this.setAttemptModel.create({
      user: Types.ObjectId.createFromHexString(studentId),
      session: session._id,
      topic: questionSet.topic,
      questionSet: questionSet._id,
      level: questionSet.level,
      answers,
      setScore,
      passed,
      strength,
      weakness,
      submittedAt,
      evaluatedAt: submittedAt,
    });

    if (passed && questionSet.level === session.currentLevel) {
      await this.sessionModel
        .updateOne({ _id: session._id }, { $inc: { currentLevel: 1 } })
        .exec();
    }

    return SetAttemptResponseDto.from(attempt);
  }

  /**
   * Evaluates a submitted answer against matching question data.
   *
   * @param submittedAnswer The submitted answer.
   * @param questions The question set questions.
   * @returns The evaluated answer payload.
   */
  private evaluateAnswer(
    submittedAnswer: SubmitAnswerItemDto,
    questions: Question[],
  ): Answer {
    const question = questions.find(
      (item) => item.id === submittedAnswer.questionId,
    );

    if (!question) {
      throw new BadRequestException('Question not found in question set');
    }

    if (question.type === QuestionType.MCQ) {
      return this.evaluateMcqAnswer(submittedAnswer, question);
    }

    return {
      id: `ans-${submittedAnswer.questionId}`,
      questionId: submittedAnswer.questionId,
      questionType: question.type,
      answer: submittedAnswer.answer,
      correctAnswer: question.correctOptionId ?? '',
      score: 0,
      feedback: '',
      targetConcepts: question.targetConcepts,
      strength: [],
      weakness: [],
      evaluatedBy: EvaluatedBy.AI,
    };
  }

  /**
   * Evaluates a submitted MCQ answer against matching question data.
   *
   * @param submittedAnswer The submitted answer.
   * @param question The matching question.
   * @returns The evaluated MCQ answer payload.
   */
  private evaluateMcqAnswer(
    submittedAnswer: SubmitAnswerItemDto,
    question: Question,
  ): Answer {
    const isCorrect = submittedAnswer.answer === question.correctOptionId;

    return {
      id: `ans-${submittedAnswer.questionId}`,
      questionId: submittedAnswer.questionId,
      questionType: question.type,
      answer: submittedAnswer.answer,
      correctAnswer: question.correctOptionId,
      score: isCorrect ? 1 : 0,
      feedback: isCorrect
        ? question.feedback.correct
        : question.feedback.incorrect,
      targetConcepts: question.targetConcepts,
      strength: isCorrect ? question.targetConcepts : [],
      weakness: isCorrect ? [] : question.targetConcepts,
      evaluatedBy: EvaluatedBy.System,
    };
  }

  /**
   * Calculates the average set score from evaluated answers.
   *
   * @param answers The evaluated answers.
   * @returns The average set score.
   */
  private calculateSetScore(answers: Answer[]): number {
    const totalScore = answers.reduce((sum, answer) => sum + answer.score, 0);

    return totalScore / answers.length;
  }

  /**
   * Collects unique target concepts matching a score.
   *
   * @param answers The evaluated answers.
   * @param score The answer score to collect concepts for.
   * @returns The matching unique target concepts.
   */
  private collectConceptsByScore(answers: Answer[], score: number): string[] {
    return [
      ...new Set(
        answers
          .filter((answer) => answer.score === score)
          .flatMap((answer) => answer.targetConcepts),
      ),
    ];
  }
}
