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
  QuestionSetType,
  QuestionType,
} from '../questions/schemas/question-set.schema';
import { Topic, TopicDocument } from '../topics/schemas/topic.schema';
import {
  SessionResponseDto,
  SetAttemptResponseDto,
  StartLiveSessionResponseDto,
  StartSessionResponseDto,
  SubmitAnswerItemDto,
  SubmitAnswerResponseDto,
  SubmitLiveAnswerResponseDto,
} from './dtos';
import {
  QUESTION_GENERATION_SYSTEM_PROMPT,
  WRITTEN_EVALUATION_SYSTEM_PROMPT,
  generatedLiveQuestionResponseFormat,
  writtenAnswerEvaluationResponseFormat,
  writtenAnswerEvaluationsSchema,
} from './ai/sessions-ai.constant';
import { AiModel } from './ai/sessions-ai.enum';
import {
  LiveGenerationPromptContext,
  SubmittedWrittenAnswer,
  WrittenAnswerEvaluation,
} from './ai/sessions-ai.types';
import {
  calculateAttemptScore,
  calculateEvaluationScore,
  calculateSetScore,
  collectAttemptConcepts,
  collectConceptsByScore,
  createLiveGenerationUserPrompt,
  createRecommendations,
  formatGeneratedLiveQuestionIds,
  getNextLiveQuestionType,
  hasPassingAnswers,
  parseGeneratedLiveQuestion,
  roundScore,
  validateGeneratedLiveQuestion,
} from './sessions.util';
import {
  LiveQuestion,
  LiveQuestionDocument,
  LiveQuestionStatus,
} from './schemas/live-question.schema';
import {
  LiveSession,
  LiveSessionDocument,
} from './schemas/live-session.schema';
import {
  SessionEvaluation,
  SessionEvaluationDocument,
} from './schemas/session-evaluation.schemas';
import {
  OverallEvaluation,
  Session,
  SessionDocument,
  SessionStatus,
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
    @InjectModel(LiveQuestion.name)
    private readonly liveQuestionModel: Model<LiveQuestionDocument>,
    @InjectModel(LiveSession.name)
    private readonly liveSessionModel: Model<LiveSessionDocument>,
    @InjectModel(SetAttempt.name)
    private readonly setAttemptModel: Model<SetAttemptDocument>,
    @InjectModel(SessionEvaluation.name)
    private readonly sessionEvaluationModel: Model<SessionEvaluationDocument>,
    private readonly configService: ConfigService,
  ) {
    const apiKey = this.configService.get<string>('TOGETHER_API_KEY');

    this.togetherClient = apiKey ? new Together({ apiKey }) : null;
    this.aiModel =
      this.configService.get<string>('AI_MODEL') ?? AiModel.GptOss120B;
  }

  private readonly togetherClient: Together | null;
  private readonly aiModel: string;

  /**
   * Starts a learning session for a user on a topic.
   *
   * @param topicId The topic ID to start.
   * @param studentId The authenticated student ID.
   * @returns The created session ID and level 0 question set.
   */
  async startSession(
    topicId: string,
    studentId: string,
  ): Promise<StartSessionResponseDto> {
    const topic = await this.topicModel.findById(topicId).exec();

    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    const questionSet = await this.questionSetModel
      .findOne({ topic: topic._id, level: 0 })
      .exec();

    if (!questionSet) {
      throw new NotFoundException('Question set not found');
    }

    const session = await this.sessionModel.create({
      student: Types.ObjectId.createFromHexString(studentId),
      topic: topic._id,
      currentLevel: 0,
      status: SessionStatus.Active,
      startedAt: new Date(),
    });

    return {
      sessionId: session._id.toString(),
      questionSet: QuestionSetResponseDto.from(questionSet),
    };
  }

  /**
   * Starts a live learning session for a user on a topic.
   *
   * @param topicId The topic ID to start.
   * @param studentId The authenticated student ID.
   * @returns The created live session and pending generated question.
   */
  async startLiveSession(
    topicId: string,
    studentId: string,
  ): Promise<StartLiveSessionResponseDto> {
    const topic = await this.topicModel.findById(topicId).exec();

    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    const level = 0;
    const questionType = getNextLiveQuestionType(level, []);
    const question = await this.generateLiveQuestion({
      topicSlug: topic.slug,
      topicTitle: topic.title,
      topicDescription: topic.description,
      topicTags: topic.tags,
      level,
      questionNumber: 1,
      questionType,
      acceptedQuestionPrompts: [],
    });
    const liveSession = await this.liveSessionModel.create({
      student: Types.ObjectId.createFromHexString(studentId),
      topic: topic._id,
      currentLevel: level,
      status: SessionStatus.Active,
      startedAt: new Date(),
    });
    const pendingQuestion = await this.liveQuestionModel.create({
      liveSession: liveSession._id,
      question,
      level,
      questionNumber: 1,
      status: LiveQuestionStatus.Pending,
    });

    return {
      sessionId: liveSession._id.toString(),
      questionId: pendingQuestion._id.toString(),
      question,
    };
  }

  /**
   * Rejects a pending live question and generates a replacement.
   *
   * @param studentId The authenticated student ID.
   * @param sessionId The live session ID.
   * @param questionId The live question document ID to reject.
   * @param reason The reason the question was rejected.
   * @returns The live session and replacement pending question.
   */
  async rejectLiveQuestion(
    studentId: string,
    sessionId: string,
    questionId: string,
    reason: string,
  ): Promise<StartLiveSessionResponseDto> {
    const liveSession = await this.liveSessionModel
      .findOne({
        _id: Types.ObjectId.createFromHexString(sessionId),
        student: Types.ObjectId.createFromHexString(studentId),
        status: SessionStatus.Active,
      })
      .exec();

    if (!liveSession) {
      throw new NotFoundException('Live session not found');
    }

    const rejectedQuestion = await this.liveQuestionModel
      .findOne({
        _id: Types.ObjectId.createFromHexString(questionId),
        liveSession: liveSession._id,
        status: LiveQuestionStatus.Pending,
      })
      .exec();

    if (!rejectedQuestion) {
      throw new NotFoundException('Live question not found');
    }

    const topic = await this.topicModel.findById(liveSession.topic).exec();

    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    const acceptedLiveQuestions = await this.liveQuestionModel
      .find({
        liveSession: liveSession._id,
        level: liveSession.currentLevel,
        status: LiveQuestionStatus.Accepted,
      })
      .exec();
    const acceptedQuestions = acceptedLiveQuestions.map(
      (liveQuestion) => liveQuestion.question,
    );
    const questionType = getNextLiveQuestionType(
      liveSession.currentLevel,
      acceptedQuestions,
    );
    const question = await this.generateLiveQuestion({
      topicSlug: topic.slug,
      topicTitle: topic.title,
      topicDescription: topic.description,
      topicTags: topic.tags,
      level: liveSession.currentLevel,
      questionNumber: acceptedQuestions.length + 1,
      questionType,
      acceptedQuestionPrompts: acceptedQuestions.map(
        (question) => question.prompt,
      ),
      rejectedQuestion: rejectedQuestion.question,
      rejectionReason: reason,
    });
    const deletedQuestion = await this.liveQuestionModel
      .findOneAndDelete({
        _id: rejectedQuestion._id,
        liveSession: liveSession._id,
        status: LiveQuestionStatus.Pending,
      })
      .exec();

    if (!deletedQuestion) {
      throw new NotFoundException('Live question not found');
    }

    const replacementQuestion = await this.liveQuestionModel.create({
      liveSession: liveSession._id,
      question,
      level: liveSession.currentLevel,
      questionNumber: acceptedQuestions.length + 1,
      status: LiveQuestionStatus.Pending,
    });

    return {
      sessionId: liveSession._id.toString(),
      questionId: replacementQuestion._id.toString(),
      question,
    };
  }

  /**
   * Submits an answer to a pending live question.
   *
   * @param studentId The authenticated student ID.
   * @param sessionId The live session ID.
   * @param questionId The live question document ID.
   * @param submittedAnswer The submitted answer.
   * @returns The evaluated answers and next live question when available.
   */
  async submitLiveAnswer(
    studentId: string,
    sessionId: string,
    questionId: string,
    submittedAnswer: string,
  ): Promise<SubmitLiveAnswerResponseDto> {
    const liveSession = await this.liveSessionModel
      .findOne({
        _id: Types.ObjectId.createFromHexString(sessionId),
        student: Types.ObjectId.createFromHexString(studentId),
        status: SessionStatus.Active,
      })
      .exec();

    if (!liveSession) {
      throw new NotFoundException('Live session not found');
    }

    const liveQuestion = await this.liveQuestionModel
      .findOne({
        _id: Types.ObjectId.createFromHexString(questionId),
        liveSession: liveSession._id,
        status: LiveQuestionStatus.Pending,
      })
      .exec();

    if (!liveQuestion) {
      throw new NotFoundException('Live question not found');
    }

    const [answer] = await this.evaluateAnswers(
      [{ questionId: liveQuestion.question.id, answer: submittedAnswer }],
      [liveQuestion.question],
    );

    const updateResult = await this.liveQuestionModel
      .updateOne(
        {
          _id: liveQuestion._id,
          status: LiveQuestionStatus.Pending,
        },
        {
          $set: {
            status: LiveQuestionStatus.Accepted,
            answer,
            answeredAt: new Date(),
          },
        },
      )
      .exec();

    if (updateResult.modifiedCount === 0) {
      throw new NotFoundException('Live question not found');
    }

    const acceptedLiveQuestions = await this.liveQuestionModel
      .find({
        liveSession: liveSession._id,
        level: liveSession.currentLevel,
        status: LiveQuestionStatus.Accepted,
      })
      .sort({ createdAt: 1 })
      .exec();
    const acceptedQuestions = acceptedLiveQuestions.map(
      (item) => item.question,
    );

    if (acceptedQuestions.length > 3) {
      throw new BadRequestException('Live session already has three questions');
    }

    if (acceptedQuestions.length === 3) {
      const questionSet = await this.questionSetModel.create({
        topic: liveSession.topic,
        setType: QuestionSetType.Live,
        level: liveSession.currentLevel,
        questions: acceptedQuestions,
      });
      const answers = acceptedLiveQuestions.map((item) => item.answer);
      const { passed } = await this.createSetAttempt(
        studentId,
        liveSession._id,
        questionSet,
        answers,
      );

      if (
        passed &&
        liveSession.currentLevel > 0 &&
        liveSession.currentLevel % 10 === 0
      ) {
        await this.createSessionEvaluation(
          liveSession,
          liveSession.currentLevel,
          true,
        );
      }

      if (!passed) {
        return {
          answers,
          nextQuestion: null,
        };
      }

      if (liveSession.currentLevel >= 100) {
        await this.liveSessionModel
          .updateOne(
            { _id: liveSession._id },
            {
              $set: {
                status: SessionStatus.Completed,
                finishedAt: new Date(),
              },
            },
          )
          .exec();

        return {
          answers,
          nextQuestion: null,
        };
      }

      const nextLevel = liveSession.currentLevel + 1;
      const topic = await this.topicModel.findById(liveSession.topic).exec();

      if (!topic) {
        throw new NotFoundException('Topic not found');
      }

      const questionType = getNextLiveQuestionType(nextLevel, []);
      const question = await this.generateLiveQuestion({
        topicSlug: topic.slug,
        topicTitle: topic.title,
        topicDescription: topic.description,
        topicTags: topic.tags,
        level: nextLevel,
        questionNumber: 1,
        questionType,
        acceptedQuestionPrompts: [],
      });
      const nextLiveQuestion = await this.liveQuestionModel.create({
        liveSession: liveSession._id,
        question,
        level: nextLevel,
        questionNumber: 1,
        status: LiveQuestionStatus.Pending,
      });

      await this.liveSessionModel
        .updateOne(
          { _id: liveSession._id },
          { $set: { currentLevel: nextLevel } },
        )
        .exec();

      return {
        answers,
        nextQuestion: {
          sessionId: liveSession._id.toString(),
          questionId: nextLiveQuestion._id.toString(),
          question,
        },
      };
    }

    const topic = await this.topicModel.findById(liveSession.topic).exec();

    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    const questionType = getNextLiveQuestionType(
      liveSession.currentLevel,
      acceptedQuestions,
    );
    const question = await this.generateLiveQuestion({
      topicSlug: topic.slug,
      topicTitle: topic.title,
      topicDescription: topic.description,
      topicTags: topic.tags,
      level: liveSession.currentLevel,
      questionNumber: acceptedQuestions.length + 1,
      questionType,
      acceptedQuestionPrompts: acceptedQuestions.map(
        (question) => question.prompt,
      ),
    });
    const nextLiveQuestion = await this.liveQuestionModel.create({
      liveSession: liveSession._id,
      question,
      level: liveSession.currentLevel,
      questionNumber: acceptedQuestions.length + 1,
      status: LiveQuestionStatus.Pending,
    });

    return {
      answers: [answer],
      nextQuestion: {
        sessionId: liveSession._id.toString(),
        questionId: nextLiveQuestion._id.toString(),
        question,
      },
    };
  }

  /**
   * Continues a live learning session for a user.
   *
   * @param studentId The authenticated student ID.
   * @param sessionId The live session ID to continue.
   * @returns The current or next pending generated question.
   */
  async continueLiveSession(
    studentId: string,
    sessionId: string,
  ): Promise<StartLiveSessionResponseDto> {
    const liveSession = await this.liveSessionModel
      .findOne({
        _id: Types.ObjectId.createFromHexString(sessionId),
        student: Types.ObjectId.createFromHexString(studentId),
        status: SessionStatus.Active,
      })
      .exec();

    if (!liveSession) {
      throw new NotFoundException('Live session not found');
    }

    const pendingQuestion = await this.liveQuestionModel
      .findOne({
        liveSession: liveSession._id,
        level: liveSession.currentLevel,
        status: LiveQuestionStatus.Pending,
      })
      .sort({ createdAt: 1 })
      .exec();

    if (pendingQuestion) {
      return {
        sessionId: liveSession._id.toString(),
        questionId: pendingQuestion._id.toString(),
        question: pendingQuestion.question,
      };
    }

    const acceptedLiveQuestions = await this.liveQuestionModel
      .find({
        liveSession: liveSession._id,
        level: liveSession.currentLevel,
        status: LiveQuestionStatus.Accepted,
      })
      .sort({ createdAt: 1 })
      .exec();
    const acceptedQuestions = acceptedLiveQuestions.map(
      (liveQuestion) => liveQuestion.question,
    );

    if (acceptedQuestions.length >= 3) {
      throw new BadRequestException('Live session already has three questions');
    }

    const topic = await this.topicModel.findById(liveSession.topic).exec();

    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    const questionType = getNextLiveQuestionType(
      liveSession.currentLevel,
      acceptedQuestions,
    );
    const question = await this.generateLiveQuestion({
      topicSlug: topic.slug,
      topicTitle: topic.title,
      topicDescription: topic.description,
      topicTags: topic.tags,
      level: liveSession.currentLevel,
      questionNumber: acceptedQuestions.length + 1,
      questionType,
      acceptedQuestionPrompts: acceptedQuestions.map(
        (question) => question.prompt,
      ),
    });
    const nextLiveQuestion = await this.liveQuestionModel.create({
      liveSession: liveSession._id,
      question,
      level: liveSession.currentLevel,
      questionNumber: acceptedQuestions.length + 1,
      status: LiveQuestionStatus.Pending,
    });

    return {
      sessionId: liveSession._id.toString(),
      questionId: nextLiveQuestion._id.toString(),
      question,
    };
  }

  /**
   * Fetches in-progress sessions for a user.
   *
   * @param studentId The authenticated student ID.
   * @returns The user's in-progress sessions.
   */
  async getInProgressSessions(
    studentId: string,
  ): Promise<SessionResponseDto[]> {
    const sessions = await this.sessionModel
      .find({
        student: Types.ObjectId.createFromHexString(studentId),
        status: SessionStatus.Active,
      })
      .populate('topic')
      .sort({ updatedAt: -1 })
      .exec();

    return SessionResponseDto.fromMany(sessions);
  }

  /**
   * Deletes a learning session owned by a user.
   *
   * @param sessionId The session ID to delete.
   * @param studentId The authenticated student ID.
   */
  async deleteSession(sessionId: string, studentId: string): Promise<void> {
    const session = await this.sessionModel
      .findOneAndDelete({
        _id: Types.ObjectId.createFromHexString(sessionId),
        student: Types.ObjectId.createFromHexString(studentId),
      })
      .exec();

    if (!session) {
      throw new NotFoundException('Session not found');
    }
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
        topic: session.topic,
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
    let nextQuestionSet: QuestionSetResponseDto | null = null;
    const attempt = await this.createSetAttempt(
      studentId,
      session._id,
      questionSet,
      answers,
    );
    const { passed } = attempt;

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
   * Creates a completed set attempt using the standard evaluation aggregates.
   *
   * @param studentId The authenticated student ID.
   * @param sessionId The regular or live session ID.
   * @param questionSet The completed question set.
   * @param answers The evaluated answers for the completed set.
   * @returns The created set attempt.
   */
  private async createSetAttempt(
    studentId: string,
    sessionId: Types.ObjectId,
    questionSet: QuestionSetDocument,
    answers: Answer[],
  ): Promise<SetAttemptDocument> {
    const setScore = calculateSetScore(answers);
    const passed = hasPassingAnswers(answers);
    const strengths = collectConceptsByScore(answers, 1);
    const weaknesses = collectConceptsByScore(answers, 0);
    const submittedAt = new Date();

    return this.setAttemptModel.create({
      user: Types.ObjectId.createFromHexString(studentId),
      session: sessionId,
      topic: questionSet.topic,
      questionSet: questionSet._id,
      level: questionSet.level,
      answers,
      setScore,
      passed,
      strengths,
      weaknesses,
      submittedAt,
      evaluatedAt: submittedAt,
    });
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
        topic: session.topic,
        level: completedLevel + 1,
      })
      .exec();

    if (!nextQuestionSet) {
      return null;
    }

    return QuestionSetResponseDto.from(nextQuestionSet);
  }

  /**
   * Generates one live-session question with AI.
   *
   * @param context The live generation prompt context.
   * @returns The generated question.
   */
  private async generateLiveQuestion(
    context: LiveGenerationPromptContext,
  ): Promise<Question> {
    if (!this.togetherClient) {
      throw new ServiceUnavailableException('AI is not configured');
    }

    const completion = await this.togetherClient.chat.completions.create({
      model: this.aiModel,
      response_format: generatedLiveQuestionResponseFormat,
      temperature: 0.7,
      messages: [
        {
          role: 'system',
          content: QUESTION_GENERATION_SYSTEM_PROMPT,
        },
        {
          role: 'user',
          content: createLiveGenerationUserPrompt(context),
        },
      ],
    });
    const content = completion.choices[0]?.message?.content;
    const text = this.extractCompletionText(content);
    const generatedQuestion = parseGeneratedLiveQuestion(text);
    const question = formatGeneratedLiveQuestionIds(
      generatedQuestion.question,
      context,
    );

    validateGeneratedLiveQuestion(question, context.questionType);

    return question;
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
      correctAnswer: evaluation.correctAnswer,
      score: roundScore(evaluation.score),
      feedback: evaluation.feedback,
      targetConcepts: writtenAnswer.question.targetConcepts,
      strengths: evaluation.strengths,
      weaknesses: evaluation.weaknesses,
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
      model: this.aiModel,
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
        keyPoints: question.rubrics.keyPoints,
        misconceptions: question.rubrics.misconceptions,
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
    if (!question.correctOptionId) {
      throw new BadRequestException('MCQ correct option not found');
    }

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
      strengths: isCorrect ? question.targetConcepts : [],
      weaknesses: isCorrect ? [] : question.targetConcepts,
      evaluatedBy: EvaluatedBy.System,
    };
  }

  /**
   * Creates a session evaluation for a completed level range.
   *
   * @param session The session to evaluate.
   * @param toLevel The last level in the evaluated range.
   * @param isLiveSession Whether the session is a live session.
   */
  private async createSessionEvaluation(
    session: SessionDocument | LiveSessionDocument,
    toLevel: number,
    isLiveSession = false,
  ): Promise<void> {
    const fromLevel = toLevel === 10 ? 0 : toLevel - 9;
    const attempts = await this.setAttemptModel
      .find({
        session: session._id,
        level: { $gte: fromLevel, $lte: toLevel },
      })
      .exec();
    const overallScore = calculateAttemptScore(attempts);
    const strengths = collectAttemptConcepts(attempts, 'strengths');
    const weaknesses = collectAttemptConcepts(attempts, 'weaknesses');
    const recommendations = createRecommendations(weaknesses);
    const summary = this.createEvaluationSummary(
      fromLevel,
      toLevel,
      overallScore,
      weaknesses,
    );

    await this.sessionEvaluationModel.create({
      student: session.student,
      session: session._id,
      topic: session.topic,
      fromLevel,
      toLevel,
      overallScore,
      summary,
      strengths,
      weaknesses,
      recommendations,
      attemptIds: attempts.map((attempt) => attempt._id.toString()),
    });

    await this.updateOverallEvaluation(session, isLiveSession);
  }

  /**
   * Updates the overall evaluation for a session.
   *
   * @param session The session to update.
   * @param isLiveSession Whether the session is a live session.
   */
  private async updateOverallEvaluation(
    session: SessionDocument | LiveSessionDocument,
    isLiveSession = false,
  ): Promise<void> {
    const evaluations = await this.sessionEvaluationModel
      .find({ session: session._id })
      .exec();
    const overallEvaluation: OverallEvaluation = {
      summary: this.createOverallSummary(evaluations),
      strengths: [
        ...new Set(evaluations.flatMap((evaluation) => evaluation.strengths)),
      ],
      weaknesses: [
        ...new Set(evaluations.flatMap((evaluation) => evaluation.weaknesses)),
      ],
      recommendations: [
        ...new Set(
          evaluations.flatMap((evaluation) => evaluation.recommendations),
        ),
      ],
    };

    if (isLiveSession) {
      await this.liveSessionModel
        .updateOne({ _id: session._id }, { $set: { overallEvaluation } })
        .exec();
    } else {
      await this.sessionModel
        .updateOne({ _id: session._id }, { $set: { overallEvaluation } })
        .exec();
    }
  }

  /**
   * Creates a summary for a level range evaluation.
   *
   * @param fromLevel The first evaluated level.
   * @param toLevel The last evaluated level.
   * @param overallScore The average score for the range.
   * @param weaknesses The weak concepts in the range.
   * @returns The generated summary.
   */
  private createEvaluationSummary(
    fromLevel: number,
    toLevel: number,
    overallScore: number,
    weaknesses: string[],
  ): string {
    if (weaknesses.length === 0) {
      return `Completed levels ${fromLevel}-${toLevel} with ${overallScore} score.`;
    }

    return `Completed levels ${fromLevel}-${toLevel} with ${overallScore} score. Review ${weaknesses.join(', ')}.`;
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
