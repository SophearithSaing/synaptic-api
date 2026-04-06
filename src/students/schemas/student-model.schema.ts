import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document } from 'mongoose';

export type StudentModelDocument = StudentModel & Document;

@Schema({ timestamps: true })
export class StudentModel {
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
