---
name: address-pr-comments
description: Fetches, analyzes, and fixes review comments on a GitHub Pull Request using GitHub CLI. Use when you need to programmatically address reviewer feedback.
---

# Address PR Comments

This skill automates the process of retrieving and resolving feedback from a GitHub Pull Request.

## Workflow

1.  **Retrieve:** Use `gh api /repos/:owner/:repo/pulls/:number/comments` to fetch all line-specific review comments.
2.  **Triage:**
    - If there are **more than 3 comments**, you MUST use `enter_plan_mode` to group the fixes and explain the strategy.
    - Identify "suggestions" (Markdown blocks) provided by the reviewer.
3.  **Implement:**
    - Apply the requested changes to the local files.
    - If you disagree with a comment, explain why in the plan or conversation.
4.  **Verify:**
    - Run `npm run build` or project-specific lint/test commands to ensure no regressions.
    - Run `npx prettier --write` on all modified files.
5.  **Review:**
    - Notify user and explicitly state that implementation and verification are complete.
    - **Do NOT** output the code changes or diffs in the chat (neither manually nor via `git diff`), as the user has already monitored the progress.
    - **STOP and wait** for explicit approval before proceeding to the commit phase.
6.  **Commit:**
    - Use **Atomic Commits** for unrelated groups of fixes.
    - Follow `GEMINI.md` commit guidelines (Capitalized, meaningful, no prefixes like 'Fix:').

## Command Reference

- Fetch comments: `gh api /repos/{owner}/{repo}/pulls/{number}/comments`

## Constraints

- **Strict Adherence:** Follow all `GEMINI.md` standards for code style, documentation (TS Doc), and shell syntax.
- **Safety:** Always verify that the `gh` CLI is authenticated before starting.
