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

- **Models:** Gemini 3 Flash and Claude 4.6 Opus.
- **Generator:** Selected model will perform the function.
- **Evaluator:** Selected model will perform the function.
- **Validation:** Manual JSON parsing + Zod schema validation for all LLM outputs.

## AI Persona

- **Student Level (P):** The session difficulty scales directly with the student's overall level (0-100).
- **Question Mix:**
  - If $P < 40$: Provide 3 MCQs.
  - If $40 \le P < 70$: Provide 1 MCQ and 2 Written questions.
  - If $P \ge 70$: Provide 3 Written questions with high technical complexity.

## Mastery System

- **Topic Mastery:** Incremented by $\lfloor \text{score} / 20 \rfloor$ after each session (max 100).
- **Overall Level:** Calculated as the average of all topic masteries, starting at 1.

## Code Style

- **Indentation:** 2 spaces (no tabs).
- **Quotes:** Use single quotes `'` for strings unless double quotes are required for JSON.
- **Line Width:** Keep code blocks under 80 characters per line where possible.
- **Comment:** Every function should contains appropriate TS Doc but limit the amount of comment inside function.

## Environment

- **Shell:** PowerShell. Use PowerShell-compatible syntax (e.g., `;` instead of `&&` for command chaining).

## Commit Standards

- **Formatting:** Edited files must be formatted with `npx prettier --write` before any commit.
- **Atomic Commits:** Separate changes into multiple logical commits. Avoid bulk commits of unrelated changes.
- **Message Quality:** Use meaningful and descriptive commit messages that explain the "why" and "what".
- **Formatting Rules:**
  - Commit messages must start with a capitalized word.
  - Do not use prefixes like 'Fix:', 'Enhancement:', or 'feat:'.
  - Avoid vague messages like "Address comments", "Small fixes", or "Update files".
