import { Controller, Post, Body } from '@nestjs/common';
import { Throttle } from '@nestjs/throttler';
import { AuthResponse, AuthService } from './auth.service';
import { LoginDto, RegisterDto } from './dto/auth.dto';

@Controller('auth')
export class AuthController {
  constructor(private readonly authService: AuthService) {}

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
  @Throttle({
    default: { limit: 5, ttl: 60000, blockDuration: 300000 },
  })
  async login(@Body() authDto: LoginDto): Promise<AuthResponse> {
    return this.authService.login(authDto.email, authDto.password);
  }
}
