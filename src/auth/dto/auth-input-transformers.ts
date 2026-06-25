import { TransformFnParams } from 'class-transformer';

/**
 * Trims a string input value.
 *
 * @param params Class transformer callback parameters.
 * @returns Trimmed string, or the original value for non-string input.
 */
export function trimInput(params: TransformFnParams): unknown {
  return typeof params.value === 'string' ? params.value.trim() : params.value;
}

/**
 * Trims and lowercases an email input value.
 *
 * @param params Class transformer callback parameters.
 * @returns Normalized email, or the original value for non-string input.
 */
export function normalizeEmailInput(params: TransformFnParams): unknown {
  return typeof params.value === 'string'
    ? params.value.trim().toLowerCase()
    : params.value;
}
