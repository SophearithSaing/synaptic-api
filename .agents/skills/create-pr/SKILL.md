---
name: create-pr
description: Creates a Pull Request using GitHub CLI, drafting a detailed description based on changes. Use when finishing a feature or fix to submit code for review.
---

# Pull Request Creator

This skill automates the creation of a Pull Request (PR) by drafting a descriptive body and invoking the GitHub CLI (`gh pr create`).

## Workflow

1. **Analyze:** Gemini CLI reviews the staged/committed changes in the current branch.
2. **Draft:** Gemini CLI drafts a detailed PR description following the project's standards:
   - **Description:** Clear summary of what was changed.
   - **Summary of Changes:**
     - Bulleted list of modifications.
     - List added changes before modified changes (new files before modified files)
   - **Technical Details:** Highlights of the approach.
   - **Verification:** Brief confirmation of build/test success.
3. **Execute:** Gemini CLI creates a temporary file for the PR description and runs `gh pr create --body-file <file>`.
4. **Cleanup:** Gemini CLI deletes the temporary description file.

## Execution

When you request to create a PR, Gemini CLI will:

1. Generate the content based on the current branch's commit history and file modifications.
2. Save this to a temporary file.
3. Execute `gh pr create --title "<Title>" --body-file <temp-file>`.
4. Delete the temporary file.

_Note: Ensure you have authenticated with `gh auth login` before using this skill._
