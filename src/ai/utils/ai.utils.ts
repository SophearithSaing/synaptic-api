/**
 * Utilities for AI-related operations.
 */

/**
 * Cleans the response text from an LLM by removing Markdown JSON blocks.
 * @param text The raw response text from the LLM.
 * @returns The cleaned JSON string.
 */
export function cleanJsonResponse(text: string): string {
  const match =
    text.match(/```json\s*([\s\S]*?)\s*```/) ||
    text.match(/```\s*([\s\S]*?)\s*```/);
  return match ? match[1].trim() : text.trim();
}

/**
 * Parses a JSON string after cleaning it.
 * @param text The raw response text.
 * @returns The parsed JSON object.
 */
export function parseAiJson<T>(text: string): T {
  const cleaned = cleanJsonResponse(text);
  try {
    return JSON.parse(cleaned) as T;
  } catch (error) {
    throw new Error(`Failed to parse AI JSON response: ${error.message}`);
  }
}
