import { Injectable, OnModuleInit, Logger } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { Topic, TopicDocument } from '../topics/schemas/topic.schema';
import {
  StudentModel,
  StudentModelDocument,
} from '../students/schemas/student-model.schema';

/**
 * Service to initialize the database with default topics and user data.
 */
@Injectable()
export class SeedService implements OnModuleInit {
  private readonly logger = new Logger(SeedService.name);

  constructor(
    @InjectModel(Topic.name) private readonly topicModel: Model<TopicDocument>,
    @InjectModel(StudentModel.name)
    private readonly studentModel: Model<StudentModelDocument>,
  ) {}

  /**
   * Automatically triggered on module initialization to seed the database.
   */
  async onModuleInit() {
    await this.seedTopics();
    await this.seedStudent();
  }

  /**
   * Seeds initial topics if none exist in the database.
   */
  private async seedTopics() {
    const count = await this.topicModel.countDocuments();
    if (count) {
      this.logger.log('Topics already exist, skipping seed.');
      return;
    }

    const topics = [
      {
        title: 'Memory Management',
        slug: 'memory-management',
        description: 'Understanding stack, heap, and GC.',
      },
      {
        title: 'Concurrency',
        slug: 'concurrency',
        description: 'Threads, locks, and async/await.',
      },
      {
        title: 'Data Structures',
        slug: 'data-structures',
        description: 'Arrays, Lists, Trees, and Graphs.',
      },
      {
        title: 'System Design',
        slug: 'system-design',
        description: 'Scaling, caching, and microservices.',
      },
      {
        title: 'Algorithms',
        slug: 'algorithms',
        description: 'Sorting, searching, and complexity.',
      },
    ];

    await this.topicModel.insertMany(topics);
    this.logger.log('5 topics seeded successfully.');
  }

  /**
   * Creates a default student model for development purposes.
   */
  private async seedStudent() {
    const userId = 'dev-user-123';
    const exists = await this.studentModel.findOne({ userId });

    if (exists) {
      this.logger.log(`Student model for ${userId} already exists.`);
      return;
    }

    await this.studentModel.create({
      userId,
      overallLevel: 0,
      topicMastery: new Map(),
    });

    this.logger.log(`Default student model created for ${userId} at level 0.`);
  }
}
