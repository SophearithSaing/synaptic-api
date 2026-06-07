import { Controller, Post, Body } from '@nestjs/common';
import { AuthResponse, AuthService } from './auth.service';
import { LoginDto, RegisterDto } from './dto/auth.dto';

@Controller('auth')
export class AuthController {
  /**
   * Creates an auth controller.
   *
   * @param authService The authentication service.
   */
  constructor(private readonly authService: AuthService) {}

  /**
   * Handles user registration requests.
   *
   * @param authDto Registration data.
   * @returns A JWT access token.
   */
  @Post('register')
  async register(@Body() authDto: RegisterDto): Promise<AuthResponse> {
    return this.authService.register(
      authDto.email,
      authDto.username,
      authDto.password,
    );
  }

  /**
   * Handles user login requests.
   *
   * @param authDto Login credentials.
   * @returns A JWT access token.
   */
  @Post('login')
  async login(@Body() authDto: LoginDto): Promise<AuthResponse> {
    return this.authService.login(authDto.email, authDto.password);
  }
}
