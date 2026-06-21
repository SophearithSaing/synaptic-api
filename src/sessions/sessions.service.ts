import {
  BadRequestException,
  Injectable,
  NotFoundException,
  ServiceUnavailableException,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { InjectModel } from '@nestjs/mongoose';
import { Model, Types } from 'mongoose';
import Together from 'together-ai';
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
  WRITTEN_EVALUATION_SYSTEM_PROMPT,
  writtenAnswerEvaluationResponseFormat,
  writtenAnswerEvaluationsSchema,
} from './ai/sessions-ai.constant';
import { EvaluationModel } from './ai/sessions-ai.enum';
import {
  SubmittedWrittenAnswer,
  WrittenAnswerEvaluation,
} from './ai/sessions-ai.types';
import {
  calculateAttemptScore,
  calculateEvaluationScore,
  calculateSetScore,
  collectAttemptConcepts,
  collectConceptsByScore,
  createRecommendations,
  roundScore,
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
    private readonly configService: ConfigService,
  ) {
    const apiKey = this.configService.get<string>('AI_API_KEY');

    this.togetherClient = apiKey ? new Together({ apiKey }) : null;
    this.evaluationModel =
      this.configService.get<string>('AI_MODEL') ?? EvaluationModel.GptOss120B;
  }

  private readonly togetherClient: Together | null;
  private readonly evaluationModel: string;

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

    const answers = await this.evaluateAnswers(
      submittedAnswers,
      questionSet.questions,
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
   * Evaluates submitted answers against matching question data.
   *
   * @param submittedAnswers The submitted answers.
   * @param questions The question set questions.
   * @returns The evaluated answer payloads.
   */
  private async evaluateAnswers(
    submittedAnswers: SubmitAnswerItemDto[],
    questions: Question[],
  ): Promise<Answer[]> {
    const writtenAnswers: SubmittedWrittenAnswer[] = [];
    const answers = submittedAnswers.map((submittedAnswer) => {
      const question = this.findQuestion(questions, submittedAnswer.questionId);

      if (question.type === QuestionType.MCQ) {
        return this.evaluateMcqAnswer(submittedAnswer, question);
      }

      writtenAnswers.push({ submittedAnswer, question });
      return null;
    });

    if (writtenAnswers.length === 0) {
      return answers.filter((answer): answer is Answer => answer !== null);
    }

    const evaluations = await this.getWrittenAnswerEvaluations(writtenAnswers);

    let writtenAnswerIndex = 0;

    return answers.map((answer) => {
      if (answer) {
        return answer;
      }

      const writtenAnswer = writtenAnswers[writtenAnswerIndex];
      writtenAnswerIndex += 1;

      return this.createWrittenAnswer(writtenAnswer, evaluations);
    });
  }

  /**
   * Finds a question by ID.
   *
   * @param questions The question set questions.
   * @param questionId The question ID to find.
   * @returns The matching question.
   */
  private findQuestion(questions: Question[], questionId: string): Question {
    const question = questions.find((item) => item.id === questionId);

    if (!question) {
      throw new BadRequestException('Question not found in question set');
    }

    return question;
  }

  /**
   * Creates an evaluated written answer from an AI evaluation.
   *
   * @param writtenAnswer The submitted written answer and question.
   * @param evaluations The AI evaluations.
   * @returns The evaluated written answer payload.
   */
  private createWrittenAnswer(
    writtenAnswer: SubmittedWrittenAnswer,
    evaluations: WrittenAnswerEvaluation[],
  ): Answer {
    const evaluation = evaluations.find(
      (item) => item.questionId === writtenAnswer.submittedAnswer.questionId,
    );

    if (!evaluation) {
      throw new ServiceUnavailableException('AI response was incomplete');
    }

    return {
      id: `ans-${writtenAnswer.submittedAnswer.questionId}`,
      questionId: writtenAnswer.submittedAnswer.questionId,
      questionType: writtenAnswer.question.type,
      answer: writtenAnswer.submittedAnswer.answer,
      correctAnswer: writtenAnswer.question.rubric.keyPoints.join('; '),
      score: roundScore(evaluation.score),
      feedback: evaluation.feedback,
      targetConcepts: writtenAnswer.question.targetConcepts,
      strength: evaluation.strength,
      weakness: evaluation.weakness,
      evaluatedBy: EvaluatedBy.AI,
    };
  }

  /**
   * Gets written answer evaluations from AI.
   *
   * @param writtenAnswers The submitted written answers and questions.
   * @returns The written answer evaluations.
   */
  private async getWrittenAnswerEvaluations(
    writtenAnswers: SubmittedWrittenAnswer[],
  ): Promise<WrittenAnswerEvaluation[]> {
    if (!this.togetherClient) {
      throw new ServiceUnavailableException('AI is not configured');
    }

    const completion = await this.togetherClient.chat.completions.create({
      model: this.evaluationModel,
      response_format: writtenAnswerEvaluationResponseFormat,
      temperature: 0,
      messages: [
        {
          role: 'system',
          content: WRITTEN_EVALUATION_SYSTEM_PROMPT,
        },
        {
          role: 'user',
          content: this.createWrittenEvaluationUserPrompt(writtenAnswers),
        },
      ],
    });
    const content = completion.choices[0]?.message?.content;
    const text = this.extractCompletionText(content);

    return this.parseWrittenAnswerEvaluations(text);
  }

  /**
   * Creates the user prompt for written answer evaluation.
   *
   * @param writtenAnswers The submitted written answers and questions.
   * @returns The user prompt.
   */
  private createWrittenEvaluationUserPrompt(
    writtenAnswers: SubmittedWrittenAnswer[],
  ): string {
    return JSON.stringify({
      answers: writtenAnswers.map(({ submittedAnswer, question }) => ({
        questionId: submittedAnswer.questionId,
        prompt: question.prompt,
        targetConcepts: question.targetConcepts,
        keyPoints: question.rubric.keyPoints,
        misconceptions: question.rubric.misconceptions,
        studentAnswer: submittedAnswer.answer,
      })),
    });
  }

  /**
   * Extracts completion text from an AI message content value.
   *
   * @param content The completion message content.
   * @returns The extracted completion text.
   */
  private extractCompletionText(content: unknown): string {
    if (typeof content === 'string') {
      return content;
    }

    throw new ServiceUnavailableException('AI response was empty');
  }

  /**
   * Parses written answer evaluations from completion text.
   *
   * @param text The completion text to parse.
   * @returns The parsed written answer evaluations.
   */
  private parseWrittenAnswerEvaluations(
    text: string,
  ): WrittenAnswerEvaluation[] {
    const normalizedText = text
      .trim()
      .replace(/^```json\s*/i, '')
      .replace(/^```\s*/i, '')
      .replace(/```$/i, '')
      .trim();
    let json: unknown;

    try {
      json = JSON.parse(normalizedText);
    } catch {
      throw new ServiceUnavailableException('AI response was invalid');
    }

    const parsed = writtenAnswerEvaluationsSchema.safeParse(json);

    if (!parsed.success) {
      throw new ServiceUnavailableException('AI response was invalid');
    }

    return parsed.data.evaluations;
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
