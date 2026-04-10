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

    let studentModel = await this.studentModel.findOne({ userId });
    if (!studentModel) {
      studentModel = await this.studentModel.create({
        userId,
        overallLevel: 1,
      });
    }

    const difficulty = studentModel.overallLevel;
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
      difficulty: 1,
      score: 0,
    });

    let progress = await this.topicProgressModel.findOne({
      userId: userId,
      topic: new Types.ObjectId(topicId),
    } as QueryFilter<TopicProgress>);
    if (!progress) {
      progress = await this.topicProgressModel.create({
        userId,
        topic: new Types.ObjectId(topicId),
        status: TopicStatus.InProgress,
        history: [questionSet._id],
      });
    } else {
      progress.history.push(questionSet._id);

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
        questionId: question._id.toString(),
        questionText: question.text,
        studentAnswer: answer.studentAnswer,
      };
    });

    const evaluation = await this.aiService.evaluateAnswers(
      evaluationData,
      provider,
    );

    // Save per-question results
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

    // Persist student answers and evaluations to individual question documents
    await Promise.all(questions.map((q) => q.save()));

    // Update session results
    questionSet.score = evaluation.totalScore;
    questionSet.feedback = evaluation.critique;
    questionSet.weakConcepts = evaluation.weakConcepts;
    questionSet.strongConcepts = evaluation.strongConcepts;
    await questionSet.save();

    // Update user's progress in StudentModel
    let studentModel = await this.studentModel.findOne({
      userId: questionSet.userId,
    });
    if (!studentModel) {
      studentModel = await this.studentModel.create({
        userId: questionSet.userId,
        overallLevel: 1,
      });
    }

    // Update topic mastery using the increment logic
    const topicId = (questionSet.topic as Types.ObjectId).toString();
    const masteryIncrement = Math.floor(evaluation.totalScore / 20);
    const currentMastery = studentModel.topicMastery.get(topicId) || 0;

    studentModel.topicMastery.set(
      topicId,
      Math.min(100, currentMastery + masteryIncrement),
    );

    // Calculate overall level as the average of all topic masteries
    const masteries = Array.from(studentModel.topicMastery.values());
    if (masteries.length > 0) {
      const averageMastery =
        masteries.reduce((sum, m) => sum + m, 0) / masteries.length;
      studentModel.overallLevel = Math.max(1, Math.round(averageMastery));
    }

    await studentModel.save();

    return {
      evaluation,
      studentModel,
    };
  }
}
