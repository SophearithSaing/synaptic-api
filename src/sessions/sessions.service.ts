import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model, QueryFilter, Types } from 'mongoose';
import { AiService } from '../ai/ai.service';
import { AiProvider } from '../ai/types/ai.types';
import {
  Question,
  QuestionDocument,
} from '../questions/schemas/question.schema';
import {
  QuestionSet,
  QuestionSetDocument,
} from '../questions/schemas/question-set.schema';
import {
  TopicProgress,
  TopicProgressDocument,
  TopicStatus,
} from '../topics/schemas/topic-progress.schema';
import { Topic, TopicDocument } from '../topics/schemas/topic.schema';
import {
  StudentModel,
  StudentModelDocument,
} from '../students/schemas/student-model.schema';

/**
 * Service to manage learning sessions, including question generation
 * and session tracking for users and topics.
 */
@Injectable()
export class SessionsService {
  constructor(
    private readonly aiService: AiService,
    @InjectModel(Question.name) private questionModel: Model<QuestionDocument>,
    @InjectModel(QuestionSet.name)
    private questionSetModel: Model<QuestionSetDocument>,
    @InjectModel(TopicProgress.name)
    private topicProgressModel: Model<TopicProgressDocument>,
    @InjectModel(Topic.name) private topicModel: Model<TopicDocument>,
    @InjectModel(StudentModel.name)
    private studentModel: Model<StudentModelDocument>,
  ) {}

  /**
   * Starts a new learning session for a specific topic and user.
   * Generates questions using AI and updates the user's progress.
   * @param topicId The ID of the topic to start.
   * @param userId The ID of the user starting the session.
   * @param provider Optional AI provider.
   * @returns The created and populated QuestionSet object.
   * @throws NotFoundException if the topic does not exist.
   */
  async startSession(topicId: string, userId: string, provider?: AiProvider) {
    const topic = await this.topicModel.findById(topicId);
    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    const generatedQuestions = await this.aiService.generateQuestions(
      topic.title,
      topic.description,
      1,
      3,
      provider,
    );
    const savedQuestions =
      await this.questionModel.insertMany(generatedQuestions);
    const questionSet = await this.questionSetModel.create({
      userId,
      topic: new Types.ObjectId(topicId),
      questions: savedQuestions.map((q) => q._id),
      difficulty: 1,
      score: 0,
    });

    let progress = await this.topicProgressModel.findOne({
      userId: userId,
      topic: topicId,
    } as QueryFilter<TopicProgress>);
    if (!progress) {
      progress = await this.topicProgressModel.create({
        userId,
        topic: new Types.ObjectId(topicId),
        status: TopicStatus.InProgress,
        history: [questionSet._id],
      });
    } else {
      progress.history.push(questionSet);

      await progress.save();
    }

    return questionSet.populate(['topic', 'questions']);
  }

  /**
   * Submits student answers for a session, evaluates them using AI,
   * updates the QuestionSet, and adjusts the student's overall level.
   * @param id The ID of the QuestionSet (session).
   * @param answers Array of question IDs and corresponding student answers.
   * @param provider Optional AI provider.
   * @returns Evaluation results and updated StudentModel.
   * @throws NotFoundException if QuestionSet or questions are not found.
   */
  async submitSession(
    id: string,
    answers: { questionId: string; studentAnswer: string }[],
    provider?: AiProvider,
  ) {
    const questionSet = await this.questionSetModel
      .findById(id)
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
        questionText: question.text,
        studentAnswer: answer.studentAnswer,
      };
    });

    // Persist student answers to individual question documents
    await Promise.all(questions.map((q) => q.save()));

    const evaluation = await this.aiService.evaluateAnswers(
      evaluationData,
      provider,
    );

    // Update session results
    questionSet.score = evaluation.totalScore;
    questionSet.feedback = evaluation.critique;
    questionSet.weakConcepts = evaluation.weakConcepts;
    questionSet.strongConcepts = evaluation.strongConcepts;
    await questionSet.save();

    // Update user's overall level in StudentModel
    let studentModel = await this.studentModel.findOne({
      userId: questionSet.userId,
    });
    if (!studentModel) {
      studentModel = await this.studentModel.create({
        userId: questionSet.userId,
        overallLevel: 1,
      });
    }

    const levelIncrement = Math.floor(evaluation.totalScore / 20);
    studentModel.overallLevel = Math.min(
      100,
      studentModel.overallLevel + levelIncrement,
    );
    await studentModel.save();

    return {
      evaluation,
      studentModel,
    };
  }
}
