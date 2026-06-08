import type { ExtensionAPI } from '@earendil-works/pi-coding-agent';

const readOnlyCommands = new Set([
  'awk',
  'basename',
  'cat',
  'cd',
  'dir',
  'dirname',
  'du',
  'echo',
  'env',
  'fd',
  'find',
  'gh',
  'git',
  'grep',
  'head',
  'less',
  'ls',
  'npm',
  'npx',
  'pwd',
  'rg',
  'sort',
  'tail',
  'test',
  'tree',
  'type',
  'wc',
  'where',
  'which',
]);

const allowedWriteCommands = new Set(['cp', 'mkdir', 'mv', 'touch']);

const readOnlyGitSubcommands = new Set([
  'branch',
  'diff',
  'grep',
  'log',
  'ls-files',
  'rev-parse',
  'show',
  'status',
]);

const allowedNpmSubcommands = new Set([
  '--version',
  '-v',
  'help',
  'list',
  'ls',
  'outdated',
  'run',
  'view',
  'whoami',
]);

const allowedNpxCommands = new Set(['prettier']);

interface PermissionRequest {
  title: string;
  reason: string;
}

interface SessionMessage {
  role?: string;
  content?: unknown;
}

interface SessionEntry {
  message?: SessionMessage;
}

interface SessionContext {
  sessionManager?: {
    getEntries(): unknown[];
  };
}

const commandRationaleInstruction = [
  'Before calling the bash tool for a command that may require permission,',
  'write one brief visible sentence explaining why that exact command is',
  'needed. Do not put this rationale only in hidden thinking.',
].join(' ');

const confirmablePatterns: RegExp[] = [
  /(^|[;&|()\s])git\s+(clean|reset|checkout|restore|switch|merge|rebase|commit|push|pull|fetch|add|rm|mv|stash|apply)\b/i,
];

const destructivePatterns: RegExp[] = [
  /(^|[;&|()\s])rm\s+/i,
  /(^|[;&|()\s])rmdir\s+/i,
  /(^|[;&|()\s])chmod\s+/i,
  /(^|[;&|()\s])chown\s+/i,
  /(^|[;&|()\s])sudo\s+/i,
  /(^|[;&|()\s])dd\s+/i,
  /(^|[;&|()\s])truncate\s+/i,
  /(^|[;&|()\s])tee\s+/i,
  /(^|[;&|()\s])sed\s+[^;&|]*\s-i(\s|$)/i,
  /(^|[;&|()\s])perl\s+[^;&|]*\s-i(\s|$)/i,
  /(^|[;&|()\s])npm\s+(install|i|ci|update|uninstall|remove|rm|exec|start|publish|link|audit\s+fix)\b/i,
  /(^|[;&|()\s])(pnpm|yarn|bun)\s+(install|add|remove|update|exec|start)\b/i,
  /(^|[;&|()\s])docker\s+(rm|rmi|run|build|compose|stop|kill|exec|volume|network)\b/i,
  /(^|[;&|()\s])kubectl\s+(apply|delete|create|replace|patch|scale|rollout|exec)\b/i,
  /(^|[;&|()\s])(mongo|mongosh|psql|mysql|redis-cli)\b/i,
  /(^|[^<>])>{1,2}[^>]/,
];

/**
 * Removes wrapping quotes from a shell token.
 *
 * @param value The shell token to normalize.
 * @returns The token without wrapping quotes.
 */
function stripQuotes(value: string): string {
  return value.replace(/^['"]|['"]$/g, '');
}

/**
 * Splits a shell command into command segments.
 *
 * @param command The shell command to split.
 * @returns The command segments.
 */
function splitSegments(command: string): string[] {
  return command
    .split(/&&|\|\||;|\|/g)
    .map((segment) => segment.trim())
    .filter(Boolean);
}

/**
 * Gets the program and arguments from a shell segment.
 *
 * @param segment The shell segment to parse.
 * @returns The parsed program and arguments.
 */
function getProgramAndArgs(segment: string): string[] {
  return segment
    .split(/\s+/g)
    .map(stripQuotes)
    .filter(Boolean)
    .filter((part) => !/^[A-Z_][A-Z0-9_]*=.*/.test(part));
}

/**
 * Checks whether a parsed shell command is allowlisted.
 *
 * @param parts The parsed command parts.
 * @returns Whether the command is allowlisted.
 */
function isAllowedProgram(parts: string[]): boolean {
  const [program, subcommand] = parts;

  if (!program) return false;

  const normalized = program.replace(/\.(exe|cmd|bat)$/i, '');

  if (!readOnlyCommands.has(normalized)) {
    return allowedWriteCommands.has(normalized);
  }

  if (normalized === 'git' && subcommand) {
    return readOnlyGitSubcommands.has(subcommand);
  }

  if (normalized === 'npm' && subcommand) {
    return allowedNpmSubcommands.has(subcommand);
  }

  if (normalized === 'npx' && subcommand) {
    return allowedNpxCommands.has(subcommand);
  }

  return true;
}

/**
 * Gets a permission request for explicitly confirmable commands.
 *
 * @param command The shell command to inspect.
 * @returns The permission request, or undefined when none is needed.
 */
function getConfirmRequest(command: string): PermissionRequest | undefined {
  const matchedPattern = confirmablePatterns.find((pattern) =>
    pattern.test(command),
  );

  if (matchedPattern) {
    return {
      title: 'Git command requires permission',
      reason: `Git command requires permission: ${matchedPattern}`,
    };
  }

  return undefined;
}

/**
 * Gets a permission request for commands blocked by the bash policy.
 *
 * @param command The shell command to inspect.
 * @returns The permission request, or undefined when the command is allowed.
 */
function getBlockedRequest(command: string): PermissionRequest | undefined {
  const matchedPattern = destructivePatterns.find((pattern) =>
    pattern.test(command),
  );

  if (matchedPattern) {
    return {
      title: 'Blocked bash command requires permission',
      reason: `Blocked potentially destructive bash command: ${matchedPattern}`,
    };
  }

  for (const segment of splitSegments(command)) {
    const parts = getProgramAndArgs(segment);

    if (!isAllowedProgram(parts)) {
      return {
        title: 'Blocked bash segment requires permission',
        reason: `Blocked non-allowlisted bash segment: ${segment}`,
      };
    }
  }

  return undefined;
}

/**
 * Checks whether a value is an object record.
 *
 * @param value The value to check.
 * @returns Whether the value is an object record.
 */
function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object';
}

/**
 * Extracts visible text from a content block.
 *
 * @param block The content block.
 * @returns The visible text in the block, or an empty string.
 */
function getBlockText(block: unknown): string {
  if (!isRecord(block)) {
    return '';
  }

  return typeof block.text === 'string' ? block.text.trim() : '';
}

/**
 * Checks whether a content block is a specific tool call.
 *
 * @param block The content block.
 * @param toolCallId The tool call id to find.
 * @returns Whether the block matches the tool call.
 */
function isMatchingToolCall(block: unknown, toolCallId: string): boolean {
  return (
    isRecord(block) &&
    block.type === 'toolCall' &&
    block.id === toolCallId &&
    block.name === 'bash'
  );
}

/**
 * Gets the visible rationale before a bash tool call.
 *
 * @param ctx The extension context.
 * @param toolCallId The tool call id.
 * @returns The command-specific rationale, or an empty string.
 */
function getToolCallRationale(ctx: SessionContext, toolCallId: string): string {
  const entries = ctx.sessionManager?.getEntries() ?? [];

  for (const entry of entries.slice().reverse()) {
    const message = (entry as SessionEntry).message;

    if (message?.role !== 'assistant' || !Array.isArray(message.content)) {
      continue;
    }

    const textBlocks: string[] = [];

    for (const block of message.content) {
      if (isMatchingToolCall(block, toolCallId)) {
        return textBlocks.join('\n').trim();
      }

      const text = getBlockText(block);

      if (text) {
        textBlocks.push(text);
      }
    }
  }

  return '';
}

/**
 * Builds the permission dialog message for a blocked command.
 *
 * @param command The requested command.
 * @param request The permission request.
 * @param rationale The command-specific rationale.
 * @returns The permission dialog message.
 */
function buildPermissionMessage(
  command: string,
  request: PermissionRequest,
  rationale: string,
): string {
  return [
    `Why it is blocked:\n${request.reason}`,
    `Why I am trying to run it:\n${
      rationale || 'No command-specific rationale was provided.'
    }`,
    `Allow this command?\n\n${command}`,
  ].join('\n\n');
}

/**
 * Registers the bash permission extension.
 *
 * @param pi The pi extension API.
 */
export default function (pi: ExtensionAPI): void {
  let permissionDenied = false;

  pi.on('before_agent_start', async (event) => {
    permissionDenied = false;

    return Promise.resolve({
      systemPrompt: `${event.systemPrompt}\n\n${commandRationaleInstruction}`,
    });
  });

  pi.on('tool_call', async (event, ctx) => {
    if (event.toolName !== 'bash') return Promise.resolve(undefined);

    const command = String((event.input.command as string) ?? '');

    if (permissionDenied) {
      return Promise.resolve({
        block: true,
        reason:
          'A previous tool permission was denied. Treating denial as a hard stop.',
      });
    }

    const request = getConfirmRequest(command) ?? getBlockedRequest(command);

    if (!request) return Promise.resolve(undefined);

    const rationale = getToolCallRationale(ctx, event.toolCallId);

    if (!ctx.hasUI) {
      return Promise.resolve({
        block: true,
        reason: `${request.reason}; no UI available. Why I tried: ${
          rationale || 'No command-specific rationale was provided.'
        }`,
      });
    }

    const allowed = await ctx.ui.confirm(
      request.title,
      buildPermissionMessage(command, request, rationale),
    );

    if (!allowed) {
      permissionDenied = true;

      return Promise.resolve({
        block: true,
        reason: 'Blocked by user. Treating denial as a hard stop.',
      });
    }

    return Promise.resolve(undefined);
  });

  pi.registerCommand('bash-permissions', {
    description: 'Show the active non-destructive bash permission policy',
    handler: (_args, ctx) => {
      ctx.ui.notify(
        'Non-destructive bash policy is active. Allowlisted read-only commands, GitHub CLI, and limited write commands are permitted. Denied permissions are treated as a hard stop.',
        'info',
      );

      return Promise.resolve();
    },
  });
}
