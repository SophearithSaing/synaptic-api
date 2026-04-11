import { Controller, Post, Body } from '@nestjs/common';
import { AuthService } from './auth.service';
import { AuthDto } from './dto/auth.dto';

@Controller('auth')
export class AuthController {
  constructor(private authService: AuthService) {}

  /**
   * Handles user registration requests.
   * @param authDto Registration data.
   */
  @Post('register')
  async register(@Body() authDto: AuthDto) {
    return this.authService.register(authDto.email, authDto.password);
  }

  /**
   * Handles user login requests.
   * @param authDto Login credentials.
   */
  @Post('login')
  async login(@Body() authDto: AuthDto) {
    return this.authService.login(authDto.email, authDto.password);
  }
}
