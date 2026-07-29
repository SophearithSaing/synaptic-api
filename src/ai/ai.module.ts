import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { RolesGuard } from '../auth/guards/roles.guard';
import { AiController } from './ai.controller';
import { AiService } from './ai.service';
import { AiLog, AiLogSchema } from './schemas/ai-log.schema';

@Module({
  imports: [
    MongooseModule.forFeature([{ name: AiLog.name, schema: AiLogSchema }]),
  ],
  controllers: [AiController],
  providers: [AiService, RolesGuard],
  exports: [AiService],
})
export class AiModule {}
