/**
 * Nest application bootstrap for fixture capture.
 *
 * Mirrors src/main.ts exactly (cookie parser, 5 MB body limits, CORS,
 * global ValidationPipe) with two harness-only differences:
 *
 * - listens on an ephemeral port (0) instead of PORT
 * - patches ThrottlerGuard.getTracker so the throttler bucket key comes
 *   from the `x-fixture-key` request header instead of the client IP.
 *   Every fixture case uses a distinct key, which isolates unrelated
 *   cases from each other (all harness traffic shares one IP);
 *   throttling itself is exercised for real by the dedicated 429
 *   scenarios, whose requests deliberately share one key.
 */
import { ValidationPipe } from '@nestjs/common';
import { NestFactory } from '@nestjs/core';
import cookieParser from 'cookie-parser';
import { ThrottlerGuard } from '@nestjs/throttler';
import { AppModule } from '../../dist/app.module.js';

export async function createCaptureApp() {
  ThrottlerGuard.prototype.getTracker = async function getTracker(req) {
    return req.headers['x-fixture-key'] ?? req.ip;
  };

  const app = await NestFactory.create(AppModule);

  app.use(cookieParser());

  app.useBodyParser('json', { limit: '5mb' });
  app.useBodyParser('urlencoded', { limit: '5mb', extended: true });

  app.enableCors({
    credentials: true,
    origin: process.env.CLIENT_URL ?? 'http://localhost:4200',
  });

  app.useGlobalPipes(
    new ValidationPipe({
      forbidNonWhitelisted: true,
      transform: true,
      whitelist: true,
    }),
  );

  await app.listen(0);

  return app;
}
