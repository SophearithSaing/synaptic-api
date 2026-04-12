import { Injectable, OnModuleInit, Logger } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { Topic, TopicDocument } from '../topics/schemas/topic.schema';
import {
  TopicCategory,
  TopicCategoryDocument,
} from '../topics/schemas/topic-category.schema';
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
    @InjectModel(TopicCategory.name)
    private readonly categoryModel: Model<TopicCategoryDocument>,
    @InjectModel(StudentModel.name)
    private readonly studentModel: Model<StudentModelDocument>,
  ) {}

  /**
   * Automatically triggered on module initialization to seed the database.
   */
  async onModuleInit(): Promise<void> {
    await this.seedCategories();
    await this.seedTopics();
    await this.seedStudent();
  }

  /**
   * Seeds initial topic categories if none exist in the database.
   */
  private async seedCategories(): Promise<void> {
    const count = await this.categoryModel.countDocuments();
    if (count) {
      this.logger.log('Categories already exist, skipping seed.');
      return;
    }

    const categories = [
      {
        title: 'Computer Science Concepts',
        slug: 'cs-concepts',
        description: 'Core theories and fundamental CS principles.',
        icon: 'cs-concepts',
      },
      {
        title: 'Languages & Tech Stacks',
        slug: 'tech-stacks',
        description: 'Modern programming languages and framework ecosystems.',
        icon: 'tech-stacks',
      },
      {
        title: 'Operations & Infrastructure',
        slug: 'ops-infra',
        description: 'DevOps, cloud architecture, and systems management.',
        icon: 'ops-infra',
      },
    ];

    await this.categoryModel.insertMany(categories);
    this.logger.log('3 categories seeded successfully.');
  }

  /**
   * Seeds initial topics associated with their respective categories.
   */
  private async seedTopics(): Promise<void> {
    const count = await this.topicModel.countDocuments();
    if (count) {
      this.logger.log('Topics already exist, skipping seed.');
      return;
    }

    const csCategory = await this.categoryModel.findOne({
      slug: 'cs-concepts',
    });
    const techCategory = await this.categoryModel.findOne({
      slug: 'tech-stacks',
    });
    const opsCategory = await this.categoryModel.findOne({ slug: 'ops-infra' });

    if (!csCategory || !techCategory || !opsCategory) {
      this.logger.error('Categories not found, cannot seed topics.');
      return;
    }

    const topics = [
      {
        title: 'Memory Management',
        slug: 'memory-management',
        description: 'Understanding stack, heap, and garbage collection.',
        icon: 'memory-management',
        category: csCategory._id,
      },
      {
        title: 'Concurrency',
        slug: 'concurrency',
        description: 'Threads, locks, and asynchronous patterns.',
        icon: 'concurrency',
        category: csCategory._id,
      },
      {
        title: 'Computer Networking',
        slug: 'computer-networking',
        description: 'OSI model, TCP/IP, and network protocols.',
        icon: 'computer-networking',
        category: csCategory._id,
      },
      {
        title: 'Node.js',
        slug: 'nodejs',
        description: 'Event loop, V8 engine, and server-side runtimes.',
        icon: 'nodejs',
        category: techCategory._id,
      },
      {
        title: 'Go',
        slug: 'go-lang',
        description: 'Statically typed, compiled language for systems.',
        icon: 'go-lang',
        category: techCategory._id,
      },
      {
        title: 'Hyperledger Fabric',
        slug: 'hyperledger-fabric',
        description: 'Permissioned blockchain framework for enterprise.',
        icon: 'hyperledger-fabric',
        category: techCategory._id,
      },
      {
        title: 'Containerization (Docker)',
        slug: 'docker',
        description: 'Isolation, image building, and orchestration.',
        icon: 'docker',
        category: opsCategory._id,
      },
      {
        title: 'CI/CD Pipelines',
        slug: 'cicd-pipelines',
        description: 'Automated testing, building, and deployment.',
        icon: 'cicd-pipelines',
        category: opsCategory._id,
      },
      {
        title: 'Kubernetes',
        slug: 'kubernetes',
        description: 'Orchestrating and managing containerized apps.',
        icon: 'kubernetes',
        category: opsCategory._id,
      },
    ];

    await this.topicModel.insertMany(topics);
    this.logger.log('9 topics seeded successfully across 3 categories.');
  }

  /**
   * Creates a default student model for development purposes.
   */
  private async seedStudent(): Promise<void> {
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
