---
name: commit-code
description: Prepare and create project commits. Use when the user asks Codex to commit code, start committing changes, make atomic commits, or stage and commit work in this repository.
---

# Commit Code

Create commits that follow the repository standards and avoid mixing unrelated
work.

## Workflow

1. Inspect the working tree:
   - Run `git status --short`.
   - Review relevant unstaged and staged diffs with `git diff` and
     `git diff --cached`.
   - Identify files modified by the current task versus unrelated existing
     changes.

2. Prepare a commit plan before staging or committing:
   - Show the user each planned commit.
   - For each planned commit, list the files to include.
   - For each planned commit, provide the exact commit message.
   - List files that will not be committed and explain why.
   - Wait for explicit user approval before starting to stage or commit.

3. Format only after the commit plan is approved:
   - Run `npx prettier --write` on edited files before committing.
   - Re-check the diff after formatting.

4. Create atomic commits:
   - Separate unrelated logical changes into separate commits.
   - Do not bulk commit unrelated files together.
   - Do not stage unrelated user changes.
   - If a file contains mixed changes, stage only the task-related hunks when
     practical. If that is risky or unclear, ask the user how to proceed.

5. Write commit messages that follow project standards:
   - Start with a capitalized word.
   - Do not use prefixes like `Fix:`, `Enhancement:`, or `feat:`.
   - Avoid vague messages like `Address comments`, `Small fixes`, or
     `Update files`.
   - Use a meaningful message that explains the what and why.

6. Verify the result:
   - Run `git status --short` after committing.
   - Report created commit hashes and any remaining uncommitted files.

## Safety Rules

- Never revert user changes unless explicitly requested.
- Never use destructive commands such as `git reset --hard` or
  `git checkout --` unless the user clearly asks for that operation.
- Prefer non-interactive git commands.
- Do not create a Pull Request unless the user explicitly asks.
