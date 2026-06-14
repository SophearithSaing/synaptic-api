# Project: Synaptic

**Goal:** Interactive AI-driven education platform for computing theory.

## Technical Constraints

- **Frontend:** Angular 17+, SCSS (Custom Styles only, NO Tailwind).
- **Backend:** NestJS, Node.js.
- **Database:** MongoDB (Mongoose).

## Code Style

- **Indentation:** 2 spaces (no tabs).
- **Quotes:** Use single quotes `'` for strings unless double quotes are required for JSON.
- **Line Width:** Keep code blocks under 80 characters per line where possible.
- **Comment:** Every function MUST have a return type and TS Doc header. Body comments are forbidden, except for complex algorithmic logic in long functions or non-obvious workarounds for third-party bugs.
- **TS Doc Format:** Include `@param` for each parameter and `@returns` only when the function returns a value. Do not write `@returns Nothing.` for `void` functions.

```ts
/**
 * Finds an item by id.
 *
 * @param itemId Item id to find.
 * @returns Matching item, or null when no item exists.
 */
private findItem(itemId: string): Item | null {
  return null;
}

/**
 * Clears the current state.
 */
public clearState(): void {
  this.state.set(null);
}
```
