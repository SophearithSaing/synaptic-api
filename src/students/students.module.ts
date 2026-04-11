import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import {
  StudentModel,
  StudentModelSchema,
} from './schemas/student-model.schema';
import { StudentsService } from './students.service';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: StudentModel.name, schema: StudentModelSchema },
    ]),
  ],
  providers: [StudentsService],
  exports: [StudentsService],
})
export class StudentsModule {}
