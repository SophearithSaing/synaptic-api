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
import {
  SetAttemptResponseDto,
  SubmitAnswerItemDto,
  SubmitAnswerResponseDto,
} from './dtos';
import {
  calculateAttemptScore,
  calculateEvaluationScore,
  calculateSetScore,
  collectAttemptConcepts,
  collectConceptsByScore,
  createRecommendations,
} from './sessions.util';
import {
  SessionEvaluation,
  SessionEvaluationDocument,
} from './schemas/session-evaluation.schemas';
import {
  OverallEvaluation,
  Session,
  SessionDocument,
} from './schemas/session.schema';
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
    @InjectModel(SessionEvaluation.name)
    private readonly sessionEvaluationModel: Model<SessionEvaluationDocument>,
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
   * @returns The created set attempt and next question set when available.
   */
  async submitAnswer(
    studentId: string,
    sessionId: string,
    questionSetId: string,
    submittedAnswers: SubmitAnswerItemDto[],
  ): Promise<SubmitAnswerResponseDto> {
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
    const setScore = calculateSetScore(answers);
    const passed = setScore >= 0.8;
    const strength = collectConceptsByScore(answers, 1);
    const weakness = collectConceptsByScore(answers, 0);
    const submittedAt = new Date();
    let nextQuestionSet: QuestionSetResponseDto | null = null;
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

      if (session.currentLevel > 0 && session.currentLevel % 10 === 0) {
        await this.createSessionEvaluation(session, session.currentLevel);
      }
    }

    if (passed) {
      nextQuestionSet = await this.getNextQuestionSet(
        session,
        questionSet.level,
      );
    }

    return {
      attempt: SetAttemptResponseDto.from(attempt),
      nextQuestionSet,
    };
  }

  /**
   * Gets the next question set after a completed level.
   *
   * @param session The current learning session.
   * @param completedLevel The completed level.
   * @returns The next question set response, or null when none exists.
   */
  private async getNextQuestionSet(
    session: SessionDocument,
    completedLevel: number,
  ): Promise<QuestionSetResponseDto | null> {
    const nextQuestionSet = await this.questionSetModel
      .findOne({
        topic: session.topic.toString(),
        level: completedLevel + 1,
      })
      .exec();

    if (!nextQuestionSet) {
      return null;
    }

    return QuestionSetResponseDto.from(nextQuestionSet);
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
   * Creates a session evaluation for a completed level range.
   *
   * @param session The session to evaluate.
   * @param toLevel The last level in the evaluated range.
   */
  private async createSessionEvaluation(
    session: SessionDocument,
    toLevel: number,
  ): Promise<void> {
    const fromLevel = toLevel === 10 ? 0 : toLevel - 9;
    const attempts = await this.setAttemptModel
      .find({
        session: session._id,
        level: { $gte: fromLevel, $lte: toLevel },
      })
      .exec();
    const overallScore = calculateAttemptScore(attempts);
    const stength = collectAttemptConcepts(attempts, 'strength');
    const weakness = collectAttemptConcepts(attempts, 'weakness');
    const recommendation = createRecommendations(weakness);
    const summary = this.createEvaluationSummary(
      fromLevel,
      toLevel,
      overallScore,
      weakness,
    );

    await this.sessionEvaluationModel.create({
      student: session.student,
      session: session._id,
      topic: session.topic,
      fromLevel,
      toLevel,
      overallScore,
      summary,
      stength,
      weakness,
      recommendation,
      attemptIds: attempts.map((attempt) => attempt._id.toString()),
    });

    await this.updateOverallEvaluation(session);
  }

  /**
   * Updates the overall evaluation for a session.
   *
   * @param session The session to update.
   */
  private async updateOverallEvaluation(
    session: SessionDocument,
  ): Promise<void> {
    const evaluations = await this.sessionEvaluationModel
      .find({ session: session._id })
      .exec();
    const overallEvaluation: OverallEvaluation = {
      summary: this.createOverallSummary(evaluations),
      stengths: [
        ...new Set(evaluations.flatMap((evaluation) => evaluation.stength)),
      ],
      weakness: [
        ...new Set(evaluations.flatMap((evaluation) => evaluation.weakness)),
      ],
      recommendations: [
        ...new Set(
          evaluations.flatMap((evaluation) => evaluation.recommendation),
        ),
      ],
    };

    await this.sessionModel
      .updateOne({ _id: session._id }, { $set: { overallEvaluation } })
      .exec();
  }

  /**
   * Creates a summary for a level range evaluation.
   *
   * @param fromLevel The first evaluated level.
   * @param toLevel The last evaluated level.
   * @param overallScore The average score for the range.
   * @param weakness The weak concepts in the range.
   * @returns The generated summary.
   */
  private createEvaluationSummary(
    fromLevel: number,
    toLevel: number,
    overallScore: number,
    weakness: string[],
  ): string {
    if (weakness.length === 0) {
      return `Completed levels ${fromLevel}-${toLevel} with ${overallScore} score.`;
    }

    return `Completed levels ${fromLevel}-${toLevel} with ${overallScore} score. Review ${weakness.join(', ')}.`;
  }

  /**
   * Creates the overall session summary from evaluations.
   *
   * @param evaluations The session evaluations.
   * @returns The generated overall summary.
   */
  private createOverallSummary(
    evaluations: SessionEvaluationDocument[],
  ): string {
    if (evaluations.length === 0) {
      return '';
    }

    const latestLevel = Math.max(
      ...evaluations.map((evaluation) => evaluation.toLevel),
    );
    const score = calculateEvaluationScore(evaluations);

    return `Completed through level ${latestLevel} with ${score} average score.`;
  }
}
