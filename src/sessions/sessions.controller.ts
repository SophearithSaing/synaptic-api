import { Controller, UseGuards } from '@nestjs/common';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';

@Controller('sessions')
@UseGuards(JwtAuthGuard)
export class SessionsController {}
