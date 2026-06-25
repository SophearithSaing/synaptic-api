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
import { randomBytes } from 'crypto';
import type { Request, Response } from 'express';
import {
  ACCESS_TOKEN_COOKIE_NAME,
  CSRF_TOKEN_COOKIE_NAME,
  getAccessTokenClearCookieOptions,
  getAccessTokenCookieOptions,
  getCsrfTokenCookieOptions,
  getRefreshTokenClearCookieOptions,
  getRefreshTokenCookieOptions,
  REFRESH_TOKEN_COOKIE_NAME,
} from './auth-cookie';
import { AuthService, AuthSessionResponse } from './auth.service';
import { LoginDto, RegisterDto } from './dto/auth.dto';
import { JwtAuthGuard } from './guards/jwt-auth.guard';
import type {
  AuthenticatedUser,
  RequestWithUser,
} from './types/request-with-user.type';

interface AuthCookieRequest extends Request {
  cookies: Record<string, string | undefined>;
}

interface AuthStatusResponse {
  authenticated: boolean;
}

interface CsrfTokenResponse {
  csrf_token: string;
}

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
   * @param response Express response used to set cookies.
   * @returns Authentication status.
   */
  @Post('register')
  @Throttle({
    default: { limit: 3, ttl: 60000, blockDuration: 300000 },
  })
  async register(
    @Body() authDto: RegisterDto,
    @Res({ passthrough: true }) response: Response,
  ): Promise<AuthStatusResponse> {
    const authResponse = await this.authService.register(
      authDto.email,
      authDto.username,
      authDto.password,
    );

    return this.createCookieAuthResponse(response, authResponse);
  }

  /**
   * Creates a CSRF token for subsequent mutating requests.
   *
   * @param response Express response used to set cookies.
   * @returns The CSRF token value.
   */
  @Get('csrf')
  getCsrfToken(
    @Res({ passthrough: true }) response: Response,
  ): CsrfTokenResponse {
    const csrfToken = randomBytes(32).toString('base64url');
    response.cookie(
      CSRF_TOKEN_COOKIE_NAME,
      csrfToken,
      getCsrfTokenCookieOptions(),
    );

    return { csrf_token: csrfToken };
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
   * Refreshes the current auth session.
   *
   * @param request Request containing the refresh token cookie.
   * @param response Express response used to set cookies.
   * @returns Authentication status.
   */
  @Post('refresh')
  async refresh(
    @Req() request: AuthCookieRequest,
    @Res({ passthrough: true }) response: Response,
  ): Promise<AuthStatusResponse> {
    const authResponse = await this.authService.refresh(
      request.cookies[REFRESH_TOKEN_COOKIE_NAME] ?? '',
    );

    return this.createCookieAuthResponse(response, authResponse);
  }

  /**
   * Logs out the current auth session.
   *
   * @param request Request containing the refresh token cookie.
   * @param response Express response used to clear cookies.
   */
  @Post('logout')
  async logout(
    @Req() request: AuthCookieRequest,
    @Res({ passthrough: true }) response: Response,
  ): Promise<void> {
    await this.authService.logout(request.cookies[REFRESH_TOKEN_COOKIE_NAME]);
    response.clearCookie(
      ACCESS_TOKEN_COOKIE_NAME,
      getAccessTokenClearCookieOptions(),
    );
    response.clearCookie(
      REFRESH_TOKEN_COOKIE_NAME,
      getRefreshTokenClearCookieOptions(),
    );
  }

  /**
   * Handles user login requests.
   *
   * @param authDto Login identifier and password.
   * @param response Express response used to set cookies.
   * @returns Authentication status.
   */
  @Post('login')
  @Throttle({
    default: { limit: 5, ttl: 60000, blockDuration: 300000 },
  })
  async login(
    @Body() authDto: LoginDto,
    @Res({ passthrough: true }) response: Response,
  ): Promise<AuthStatusResponse> {
    const authResponse = await this.authService.login(
      authDto.identifier,
      authDto.password,
    );

    return this.createCookieAuthResponse(response, authResponse);
  }

  /**
   * Sets auth cookies and returns auth status.
   *
   * @param response Express response used to set cookies.
   * @param authResponse Auth response containing generated tokens.
   * @returns Authentication status.
   */
  private createCookieAuthResponse(
    response: Response,
    authResponse: AuthSessionResponse,
  ): AuthStatusResponse {
    response.cookie(
      ACCESS_TOKEN_COOKIE_NAME,
      authResponse.access_token,
      getAccessTokenCookieOptions(this.configService),
    );
    response.cookie(
      REFRESH_TOKEN_COOKIE_NAME,
      authResponse.refresh_token,
      getRefreshTokenCookieOptions(this.configService),
    );

    return { authenticated: true };
  }
}
