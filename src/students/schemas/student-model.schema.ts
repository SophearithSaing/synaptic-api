import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document } from 'mongoose';

export type StudentModelDocument = StudentModel & Document;

@Schema({ timestamps: true, collection: 'studentModels' })
export class StudentModel {
  @Prop({ required: true, unique: true, index: true })
  userId: string;

  @Prop({ required: true, default: 1 })
  overallLevel: number;

  @Prop({
    type: Map,
    of: Number,
    default: {},
  })
  topicMastery: Map<string, number>;
}

export const StudentModelSchema = SchemaFactory.createForClass(StudentModel);
