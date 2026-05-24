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

function stripQuotes(value: string): string {
  return value.replace(/^['"]|['"]$/g, '');
}

function splitSegments(command: string): string[] {
  return command
    .split(/&&|\|\||;|\|/g)
    .map((segment) => segment.trim())
    .filter(Boolean);
}

function getProgramAndArgs(segment: string): string[] {
  return segment
    .split(/\s+/g)
    .map(stripQuotes)
    .filter(Boolean)
    .filter((part) => !/^[A-Z_][A-Z0-9_]*=.*/.test(part));
}

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

function getConfirmReason(command: string): string | undefined {
  const matchedPattern = confirmablePatterns.find((pattern) =>
    pattern.test(command),
  );

  if (matchedPattern) {
    return `Git command requires permission: ${matchedPattern}`;
  }

  return undefined;
}

function getBlockReason(command: string): string | undefined {
  const matchedPattern = destructivePatterns.find((pattern) =>
    pattern.test(command),
  );

  if (matchedPattern) {
    return `Blocked potentially destructive bash command: ${matchedPattern}`;
  }

  for (const segment of splitSegments(command)) {
    const parts = getProgramAndArgs(segment);

    if (!isAllowedProgram(parts)) {
      return `Blocked non-allowlisted bash segment: ${segment}`;
    }
  }

  return undefined;
}

export default function (pi: ExtensionAPI): void {
  let permissionDenied = false;

  pi.on('before_agent_start', async () => {
    permissionDenied = false;

    return Promise.resolve();
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

    const confirmReason = getConfirmReason(command);

    if (confirmReason) {
      if (!ctx.hasUI) {
        return Promise.resolve({
          block: true,
          reason: `${confirmReason}; no UI available`,
        });
      }

      const allowed = await ctx.ui.confirm(
        'Git command requires permission',
        `Allow this command?\n\n${command}`,
      );

      if (!allowed) {
        permissionDenied = true;

        return Promise.resolve({
          block: true,
          reason: 'Blocked by user. Treating denial as a hard stop.',
        });
      }

      return Promise.resolve(undefined);
    }

    const reason = getBlockReason(command);

    if (!reason) return Promise.resolve(undefined);

    if (ctx.hasUI) {
      ctx.ui.notify(reason, 'warning');
    }

    return Promise.resolve({ block: true, reason });
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
