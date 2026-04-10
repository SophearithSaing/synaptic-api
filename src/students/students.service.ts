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

  async create(userId: string): Promise<StudentModel> {
    const newStudent = new this.studentModel({ userId });
    return newStudent.save();
  }
}
