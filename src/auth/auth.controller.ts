import {
  Body,
  Controller,
  Get,
  Post,
  Req,
  Res,
  UseGuards,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Throttle } from '@nestjs/throttler';
import type { Response } from 'express';
import {
  ACCESS_TOKEN_COOKIE_NAME,
  getAccessTokenCookieOptions,
} from './auth-cookie';
import { AuthResponse, AuthService } from './auth.service';
import { LoginDto, RegisterDto } from './dto/auth.dto';
import { JwtAuthGuard } from './guards/jwt-auth.guard';
import type {
  AuthenticatedUser,
  RequestWithUser,
} from './types/request-with-user.type';

@Controller('auth')
export class AuthController {
  constructor(
    private readonly authService: AuthService,
    private readonly configService: ConfigService,
  ) {}

  /**
   * Handles user registration requests.
   *
   * @param authDto Registration data.
   * @returns A JWT access token.
   */
  @Post('register')
  @Throttle({
    default: { limit: 3, ttl: 60000, blockDuration: 300000 },
  })
  async register(
    @Body() authDto: RegisterDto,
    @Res({ passthrough: true }) response: Response,
  ): Promise<AuthResponse> {
    const authResponse = await this.authService.register(
      authDto.email,
      authDto.username,
      authDto.password,
    );

    return this.createCookieAuthResponse(response, authResponse);
  }

  /**
   * Fetches the current authenticated user.
   *
   * @param request The authenticated request.
   * @returns The current authenticated user.
   */
  @Get('me')
  @UseGuards(JwtAuthGuard)
  getCurrentUser(@Req() request: RequestWithUser): AuthenticatedUser {
    return request.user;
  }

  /**
   * Handles user login requests.
   *
   * @param authDto Login identifier and password.
   * @returns A JWT access token.
   */
  @Post('login')
  @Throttle({
    default: { limit: 5, ttl: 60000, blockDuration: 300000 },
  })
  async login(
    @Body() authDto: LoginDto,
    @Res({ passthrough: true }) response: Response,
  ): Promise<AuthResponse> {
    const authResponse = await this.authService.login(
      authDto.identifier,
      authDto.password,
    );

    return this.createCookieAuthResponse(response, authResponse);
  }

  /**
   * Sets the access token cookie and returns the auth response body.
   *
   * @param response Express response used to set cookies.
   * @param authResponse Auth response containing the access token.
   * @returns The unchanged auth response body.
   */
  private createCookieAuthResponse(
    response: Response,
    authResponse: AuthResponse,
  ): AuthResponse {
    response.cookie(
      ACCESS_TOKEN_COOKIE_NAME,
      authResponse.access_token,
      getAccessTokenCookieOptions(this.configService),
    );

    return authResponse;
  }
}
