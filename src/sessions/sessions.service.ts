import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model, Types } from 'mongoose';
import { Topic, TopicDocument } from '../topics/schemas/topic.schema';
import { SessionResponseDto } from './dtos';
import { Session, SessionDocument } from './schemas/session.schema';

@Injectable()
export class SessionsService {
  constructor(
    @InjectModel(Session.name)
    private readonly sessionModel: Model<SessionDocument>,
    @InjectModel(Topic.name)
    private readonly topicModel: Model<TopicDocument>,
  ) {}

  /**
   * Starts a learning session for a user on a topic.
   *
   * @param topicId The topic ID to start.
   * @param studentId The authenticated student ID.
   * @returns The created session.
   */
  async startSession(
    topicId: string,
    studentId: string,
  ): Promise<SessionResponseDto> {
    const topic = await this.topicModel.findById(topicId).exec();

    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    const session = await this.sessionModel.create({
      student: Types.ObjectId.createFromHexString(studentId),
      topic: topic._id,
      currentLevel: 0,
      status: 'active',
      startAt: new Date(),
    });

    return SessionResponseDto.from(session);
  }
}
