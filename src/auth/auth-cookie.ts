import { ConfigService } from '@nestjs/config';
import type { CookieOptions } from 'express';

export const ACCESS_TOKEN_COOKIE_NAME = 'access_token';
export const REFRESH_TOKEN_COOKIE_NAME = 'refresh_token';

/**
 * Creates options for clearing the access token auth cookie.
 *
 * @returns Access token clear-cookie options.
 */
export function getAccessTokenClearCookieOptions(): CookieOptions {
  return {
    httpOnly: true,
    path: '/',
    sameSite: 'none',
    secure: true,
  };
}

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
    ...getAccessTokenClearCookieOptions(),
    maxAge: parseDuration(
      configService.getOrThrow<string>('JWT_ACCESS_EXPIRES_IN'),
    ),
  };
}

/**
 * Creates options for clearing the refresh token auth cookie.
 *
 * @returns Refresh token clear-cookie options.
 */
export function getRefreshTokenClearCookieOptions(): CookieOptions {
  return {
    httpOnly: true,
    path: '/auth',
    sameSite: 'none',
    secure: true,
  };
}

/**
 * Creates options for the refresh token auth cookie.
 *
 * @param configService Application configuration service.
 * @returns Refresh token cookie options.
 */
export function getRefreshTokenCookieOptions(
  configService: ConfigService,
): CookieOptions {
  return {
    ...getRefreshTokenClearCookieOptions(),
    maxAge: getRefreshTokenMaxAge(configService),
  };
}

/**
 * Gets the refresh token max age from config.
 *
 * @param configService Application configuration service.
 * @returns Refresh token max age in milliseconds.
 */
export function getRefreshTokenMaxAge(configService: ConfigService): number {
  return parseDuration(
    configService.getOrThrow<string>('JWT_REFRESH_EXPIRES_IN'),
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
