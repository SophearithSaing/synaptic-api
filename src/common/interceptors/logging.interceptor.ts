import {
  CallHandler,
  ExecutionContext,
  Injectable,
  Logger,
  NestInterceptor,
} from '@nestjs/common';
import { Request, Response } from 'express';
import { Observable, tap } from 'rxjs';

@Injectable()
export class LoggingInterceptor implements NestInterceptor {
  private readonly logger = new Logger(LoggingInterceptor.name);

  /**
   * Logs HTTP request completion, failure, and duration.
   *
   * @param context The current execution context.
   * @param next The next handler in the request pipeline.
   * @returns The handler response stream.
   */
  intercept(context: ExecutionContext, next: CallHandler): Observable<unknown> {
    const request = context.switchToHttp().getRequest<Request>();
    const response = context.switchToHttp().getResponse<Response>();
    const method = request.method;
    const url = request.originalUrl ?? request.url;
    const startedAt = Date.now();

    return next.handle().pipe(
      tap({
        next: () => {
          const duration = Date.now() - startedAt;
          this.logger.log(
            `${method} ${url} ${response.statusCode} ${duration}ms`,
          );
        },
        error: (error: Error) => {
          const duration = Date.now() - startedAt;
          const statusCode =
            response.statusCode >= 400 ? response.statusCode : 500;

          this.logger.error(
            `${method} ${url} ${statusCode} ${duration}ms - ${error.message}`,
          );
        },
      }),
    );
  }
}
