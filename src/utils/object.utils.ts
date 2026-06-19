/**
 * Determines whether a value can be converted to a plain object.
 *
 * @param value The value to inspect.
 * @returns Whether the value has a typed toObject method.
 */
export function hasToObject(value: unknown): value is { toObject(): unknown } {
  return (
    value !== null &&
    typeof value === 'object' &&
    typeof (value as { toObject?: unknown }).toObject === 'function'
  );
}
