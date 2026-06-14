/**
 * Converts an unknown error into a standard Error object.
 *
 * @param error The error to convert.
 * @returns A standard Error instance.
 */
export function toError(error: unknown): Error {
  if (error instanceof Error) {
    return error;
  }

  if (error && typeof error === 'object' && 'message' in error) {
    return new Error(String((error as { message: unknown }).message));
  }

  return new Error(String(error));
}

/**
 * Extracts the message from an unknown error.
 *
 * @param error The error to extract the message from.
 * @returns The error message string.
 */
export function getErrorMessage(error: unknown): string {
  return toError(error).message;
}
