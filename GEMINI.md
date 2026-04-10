# Project: Curiosity

**Goal:** Interactive AI-driven education platform for computing theory.
**Status:** Multi-repo Architecture.

## Technical Constraints

- **Frontend:** Angular 18+, SCSS (Custom Styles only, NO Tailwind).
- **Backend:** NestJS, Node.js.
- **Database:** MongoDB (Mongoose).
- **State:** Angular Signals for reactive UI (Progress Bar, Question State).

## UI/UX Direction

- **Branding:** Minimalist, academic, interactive.
- **Componentry:** Custom SCSS modules. Focus on high-quality typography and transitions.
- **Interaction:** No chat bubbles. Question-based cards with $0 \rightarrow 100$ progress tracking.

## AI Approach

- **Models:** Gemini 2.5/3 Flash (via `@google/generative-ai`) and Claude 4.6 (via `@anthropic-ai/sdk`).
- **Generator:** Selected model will perform the function
- **Evaluator:** Selected model will perform the function
- **Validation:** Manual JSON parsing + Zod schema validation for LLM outputs.

## AI Persona

- When generating sessions, if student level $P < 40$, provide 3 MCQs. If $40 \le P < 70$, provide 1 MCQ and 2 Written. If $P \ge 70$, provide 3 Written questions with high technical complexity."

## Code Style

- **Indentation:** 2 spaces (no tabs).
- **Quotes:** Use single quotes `'` for strings unless double quotes are required for JSON.
- **Line Width:** Keep code blocks under 80 characters per line where possible.
- **Comment:** Every function should contains appropriate TS Doc but limit the amount of comment inside function.

## Commit Standards

- **Formatting:** Edited files must be formatted with `npx prettier --write` before any commit.
- **Message:** Commit messages must start with a capitalized word.
- **Prefixes:** Do not use prefixes like 'Fix:', 'Enhancement:', or 'feat:'.
