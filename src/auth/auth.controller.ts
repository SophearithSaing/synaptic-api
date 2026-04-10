import {
  Controller,
  Post,
  Body,
  UseGuards,
  Get,
  Request as NestRequest,
} from '@nestjs/common';
import { Request } from 'express';
import { AuthService, UserObject } from './auth.service';
import { LocalAuthGuard } from './guards/local-auth.guard';
import { JwtAuthGuard } from './guards/jwt-auth.guard';
import { CurrentUser } from './decorators/current-user.decorator';

interface RegisterDto {
  email: string;
  password: string;
}

interface RefreshDto {
  refreshToken: string;
}

interface RequestWithUser extends Request {
  user: UserObject;
}

@Controller('auth')
export class AuthController {
  constructor(private authService: AuthService) {}

  /**
   * Registers a new user with the provided credentials.
   */
  @Post('register')
  async register(@Body() body: RegisterDto) {
    return this.authService.register(body.email, body.password);
  }

  @UseGuards(LocalAuthGuard)
  @Post('login')
  async login(@NestRequest() req: RequestWithUser) {
    return this.authService.login(req.user);
  }

  @Post('refresh')
  async refresh(@Body() body: RefreshDto) {
    return this.authService.refresh(body.refreshToken);
  }

  @UseGuards(JwtAuthGuard)
  @Post('logout')
  async logout(@CurrentUser() user: { userId: string; email: string }) {
    return this.authService.logout(user.userId);
  }

  @UseGuards(JwtAuthGuard)
  @Get('profile')
  getProfile(@CurrentUser() user: { userId: string; email: string }) {
    return user;
  }
}
