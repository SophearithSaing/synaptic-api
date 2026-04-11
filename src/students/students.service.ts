import { Injectable } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import {
  StudentModel,
  StudentModelDocument,
} from './schemas/student-model.schema';

@Injectable()
export class StudentsService {
  constructor(
    @InjectModel(StudentModel.name)
    private studentModel: Model<StudentModelDocument>,
  ) {}

  /**
   * Creates a new student profile for a given user.
   * @param userId The ID of the user.
   * @returns The created student document.
   */
  async create(userId: string): Promise<StudentModel> {
    const newStudent = new this.studentModel({ userId });
    return newStudent.save();
  }
}
