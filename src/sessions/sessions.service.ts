import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model, Types } from 'mongoose';
import { QuestionSetResponseDto } from '../questions/dtos';
import {
  QuestionSet,
  QuestionSetDocument,
} from '../questions/schemas/question-set.schema';
import { Topic, TopicDocument } from '../topics/schemas/topic.schema';
import { Session, SessionDocument } from './schemas/session.schema';

@Injectable()
export class SessionsService {
  constructor(
    @InjectModel(Session.name)
    private readonly sessionModel: Model<SessionDocument>,
    @InjectModel(Topic.name)
    private readonly topicModel: Model<TopicDocument>,
    @InjectModel(QuestionSet.name)
    private readonly questionSetModel: Model<QuestionSetDocument>,
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
}
