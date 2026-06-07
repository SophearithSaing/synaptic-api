import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model, Types } from 'mongoose';
import { AiService } from '../ai/ai.service';
import { AiAnswerEvaluation, AiProvider } from '../ai/types/ai.types';
import {
  Question,
  QuestionDocument,
} from '../questions/schemas/question.schema';
import {
  QuestionSet,
  QuestionSetDocument,
} from '../questions/schemas/question-set.schema';
import { Topic, TopicDocument } from '../topics/schemas/topic.schema';

/**
 * Service to manage learning sessions, including question generation
 * and session tracking for users and topics.
 */
@Injectable()
export class SessionsService {
  /**
   * Creates a sessions service.
   *
   * @param aiService The AI service.
   * @param questionModel The question model.
   * @param questionSetModel The question set model.
   * @param topicModel The topic model.
   */
  constructor(
    private readonly aiService: AiService,
    @InjectModel(Question.name) private questionModel: Model<QuestionDocument>,
    @InjectModel(QuestionSet.name)
    private questionSetModel: Model<QuestionSetDocument>,
    @InjectModel(Topic.name) private topicModel: Model<TopicDocument>,
  ) {}

  /**
   * Starts a new learning session for a specific topic and user.
   * Generates questions using AI.
   *
   * @param topicId The ID of the topic to start.
   * @param userId The ID of the user starting the session.
   * @param provider Optional AI provider.
   * @returns The created and populated question set object.
   * @throws NotFoundException if the topic does not exist.
   */
  async startSession(
    topicId: string,
    userId: string,
    provider?: AiProvider,
  ): Promise<QuestionSetDocument> {
    const topic = await this.topicModel.findById(topicId);
    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    const difficulty = 0;
    const count = 3;

    const generatedQuestions = await this.aiService.generateQuestions(
      topic.title,
      topic.description,
      difficulty,
      count,
      provider,
    );
    const savedQuestions =
      await this.questionModel.insertMany(generatedQuestions);
    const questionSet = await this.questionSetModel.create({
      userId,
      topic: new Types.ObjectId(topicId),
      questions: savedQuestions.map((q) => q._id),
      difficulty,
      score: 0,
    });

    return questionSet.populate(['topic', 'questions']);
  }

  /**
   * Submits student answers for a session and evaluates them using AI.
   *
   * @param id The ID of the question set session.
   * @param userId The ID of the user submitting the session.
   * @param answers Question IDs and corresponding student answers.
   * @param provider Optional AI provider.
   * @returns Evaluation results.
   * @throws NotFoundException if the question set or questions are not found.
   */
  async submitSession(
    id: string,
    userId: string,
    answers: { questionId: string; studentAnswer: string }[],
    provider?: AiProvider,
  ): Promise<{
    evaluation: AiAnswerEvaluation;
  }> {
    const questionSet = await this.questionSetModel
      .findOne({ _id: id, userId })
      .populate('questions');
    if (!questionSet) {
      throw new NotFoundException('Question set not found');
    }

    const questions = questionSet.questions as QuestionDocument[];
    const evaluationData = answers.map((answer) => {
      const question = questions.find(
        (q) => q._id.toString() === answer.questionId,
      );
      if (!question) {
        throw new NotFoundException(
          `Question ${answer.questionId} not found in this set`,
        );
      }
      question.studentAnswer = answer.studentAnswer;
      return {
        questionId: question._id.toString(),
        questionText: question.text,
        studentAnswer: answer.studentAnswer,
      };
    });

    const evaluation = await this.aiService.evaluateAnswers(
      evaluationData,
      provider,
    );

    evaluation.questionEvaluations.forEach((qEval) => {
      const question = questions.find(
        (q) => q._id.toString() === qEval.questionId,
      );
      if (question) {
        question.score = qEval.score;
        question.isCorrect = qEval.isCorrect;
        question.feedback = qEval.feedback;
      }
    });

    await Promise.all(questions.map((q) => q.save()));

    questionSet.score = evaluation.totalScore;
    questionSet.feedback = evaluation.critique;
    questionSet.weakConcepts = evaluation.weakConcepts;
    questionSet.strongConcepts = evaluation.strongConcepts;
    await questionSet.save();

    return {
      evaluation,
    };
  }
}
