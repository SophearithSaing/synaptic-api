import { ConfigService } from '@nestjs/config';
import type { CookieOptions } from 'express';

export const ACCESS_TOKEN_COOKIE_NAME = 'access_token';

/**
 * Creates options for the access token auth cookie.
 *
 * @param configService Application configuration service.
 * @returns Access token cookie options.
 */
export function getAccessTokenCookieOptions(
  configService: ConfigService,
): CookieOptions {
  return {
    httpOnly: true,
    maxAge: getAccessTokenMaxAge(configService),
    path: '/',
    sameSite: 'none',
    secure: true,
  };
}

/**
 * Gets the access token cookie max age from JWT expiry config.
 *
 * @param configService Application configuration service.
 * @returns Access token max age in milliseconds.
 */
function getAccessTokenMaxAge(configService: ConfigService): number {
  return parseDuration(
    configService.getOrThrow<string>('JWT_ACCESS_EXPIRES_IN'),
  );
}

/**
 * Parses a compact duration string to milliseconds.
 *
 * @param duration Duration string using ms, s, m, h, or d units.
 * @returns Duration in milliseconds.
 */
function parseDuration(duration: string): number {
  const match = duration.trim().match(/^(\d+)(ms|s|m|h|d)$/);

  if (!match) {
    throw new Error(`Invalid duration value: ${duration}`);
  }

  const value = Number(match[1]);
  const unit = match[2];
  const multipliers: Record<string, number> = {
    ms: 1,
    s: 1000,
    m: 60000,
    h: 3600000,
    d: 86400000,
  };

  return value * multipliers[unit];
}
