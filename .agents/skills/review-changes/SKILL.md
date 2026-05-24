---
name: review-changes
description: Reviews committed and uncommitted changes, separating findings into both sections with concise safety notes, concerns, and suggestions.
---

# Review Changes

Review repository changes like a lightweight code review. Compare both committed and uncommitted work, then report concise findings without being picky.

## Workflow

1. **Inspect committed changes:**
   - Use `git log` to identify relevant commits on the current branch.
   - Use `git diff` against the branch base or appropriate upstream to review committed changes.
2. **Inspect uncommitted changes:**
   - Use `git status --short` to identify working tree changes.
   - Use `git diff` and `git diff --cached` to review unstaged and staged changes.
3. **Review:**
   - Split the response into two sections:
     - `Committed Changes`
     - `Uncommitted Changes`
   - Under each section, include only meaningful findings.
   - Use exactly three finding categories:
     - `Safety`
     - `Concern`
     - `Suggestion`
   - Use `Safety` for security, permission, data loss, or destructive-action risks.
   - Keep each message short.
   - Do not be picky; skip minor style preferences unless they affect clarity, safety, correctness, or maintainability.
4. **Format findings:**
   - Use the commit hash and commit message as the heading for committed findings.
   - Use `Working Tree` as the heading for uncommitted findings.
   - Include a short code snippet for each finding inside a `diff` fenced block.
   - Show the `diff` snippet first, then the finding category and message.

## Output Format

- `## Committed Changes`
- `### <commit-hash> <commit message>`
- A short fenced `diff` snippet supporting the finding.
- `**Safety:** <short message>`, `**Concern:** <short message>`, or `**Suggestion:** <short message>`
- `## Uncommitted Changes`
- `### Working Tree`
- A short fenced `diff` snippet supporting the finding.
- `**Safety:** <short message>`, `**Concern:** <short message>`, or `**Suggestion:** <short message>`

If there are no findings in a section, write `No concerns or suggestions.`
