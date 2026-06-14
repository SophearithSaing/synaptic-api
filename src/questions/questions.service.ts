import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import {
  CreateQuestionSetDto,
  QuestionSetResponseDto,
  UpdateQuestionSetDto,
} from './dtos';
import {
  Question,
  QuestionSet,
  QuestionSetDocument,
} from './schemas/question-set.schema';
import { Topic, TopicDocument } from '../topics/schemas/topic.schema';

@Injectable()
export class QuestionsService {
  constructor(
    @InjectModel(QuestionSet.name)
    private questionSetModel: Model<QuestionSetDocument>,
    @InjectModel(Topic.name)
    private topicModel: Model<TopicDocument>,
  ) {}

  /**
   * Creates a question set.
   *
   * @param dto The question set details.
   * @returns The created question set.
   */
  async createQuestionSet(
    dto: CreateQuestionSetDto,
  ): Promise<QuestionSetResponseDto> {
    const questionSet = await this.questionSetModel.create(dto);
    return QuestionSetResponseDto.from(questionSet);
  }

  /**
   * Creates multiple question sets.
   *
   * @param dtos The question set details.
   * @returns The created question sets.
   */
  async createQuestionSets(
    dtos: CreateQuestionSetDto[],
  ): Promise<QuestionSetResponseDto[]> {
    const questionSets = await this.questionSetModel.create(dtos);
    return QuestionSetResponseDto.fromMany(questionSets);
  }

  /**
   * Updates a question set by ID.
   *
   * @param id The question set ID.
   * @param dto The question set updates.
   * @returns The updated question set.
   */
  async updateQuestionSet(
    id: string,
    dto: UpdateQuestionSetDto,
  ): Promise<QuestionSetResponseDto> {
    const questionSet = await this.questionSetModel
      .findByIdAndUpdate(id, dto, { new: true, runValidators: true })
      .exec();

    if (!questionSet) {
      throw new NotFoundException('Question set not found');
    }

    return QuestionSetResponseDto.from(questionSet);
  }

  /**
   * Fetches a question set by ID.
   *
   * @param id The question set ID.
   * @param populateTopic Whether to populate the topic.
   * @returns The requested question set.
   */
  async getQuestionSetById(
    id: string,
    populateTopic: boolean,
  ): Promise<QuestionSetResponseDto> {
    const query = this.questionSetModel.findById(id);

    if (populateTopic) {
      query.populate('topic');
    }

    const questionSet = await query.exec();

    if (!questionSet) {
      throw new NotFoundException('Question set not found');
    }

    return QuestionSetResponseDto.from(questionSet);
  }

  /**
   * Fetches question sets by topic slug.
   *
   * @param topicSlug The topic slug.
   * @param populateTopic Whether to populate the topic.
   * @returns The question sets for the topic.
   */
  async getQuestionSetsByTopicSlug(
    topicSlug: string,
    populateTopic: boolean,
  ): Promise<QuestionSetResponseDto[]> {
    const topic = await this.topicModel.findOne({ slug: topicSlug }).exec();

    if (!topic) {
      throw new NotFoundException('Topic not found');
    }

    const query = this.questionSetModel.find({ topic: topic._id });

    if (populateTopic) {
      query.populate('topic');
    }

    const questionSets = await query.exec();

    return QuestionSetResponseDto.fromMany(questionSets);
  }

  /**
   * Generates questions for a topic.
   *
   * @param topic The topic to generate questions for.
   * @returns The generated question payload.
   */
  generateQuestions(topic: TopicDocument): Question[] {
    void topic;
    throw new Error('Not implemented');
  }
}
