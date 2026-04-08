import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model, QueryFilter, Types } from 'mongoose';
import { AiService } from '../ai/ai.service';
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
  ) {}

  async startSession(topicId: string, userId: string) {
    const topic = await this.topicModel.findById(topicId);
    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    const generatedQuestions = await this.aiService.generateQuestions(
      topic.title,
      topic.description,
      1,
      3,
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
}
