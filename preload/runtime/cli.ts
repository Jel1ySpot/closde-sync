import type { CliArgs } from './types.ts';
import type { ClientConfig } from '../sync/types.ts';
import { stringValue } from './arg-utils.ts';
import { expandConfigPath } from '../infra/shared/path.ts';
import { homedir } from 'node:os';

const DEFAULT_CLAUDE_SETTINGS = `${homedir()}/.claude.json`;
const DEFAULT_CLAUDE_CREDENTIALS = `${homedir()}/.claude/.credentials.json`;

export const HELP_TEXT = `closde client

Usage:
  node --experimental-strip-types src/main.ts --server-url <url> --auth-token <token>

Flags:
  --server-url              Server base URL. Defaults to CLOSDE_SERVER_URL.
  --auth-token              Auth token for this client. Defaults to CLOSDE_AUTH_TOKEN or AUTH_TOKEN.
  --proxy-url               HTTP proxy URL. Defaults to CLOSDE_PROXY.
  --platform                Override process.platform. Defaults to CLOSDE_PLATFORM or current process.platform.
  --arch                    Override process.arch. Defaults to CLOSDE_ARCH or current process.arch.
  --node-version            Override Node version. Defaults to CLOSDE_NODE_VERSION or current process.version.
  --claude-settings         Claude settings path. Defaults to CLOSDE_CLAUDE_SETTINGS or ${DEFAULT_CLAUDE_SETTINGS}.
  --claude-credentials      Claude credentials path. Defaults to CLOSDE_CLAUDE_CREDENTIALS or ${DEFAULT_CLAUDE_CREDENTIALS}.
  --validate                Parse configuration and exit.
  --help                    Show help.
`;

export function loadConfig(args: CliArgs): ClientConfig {
  const serverUrl = stringValue(args['server-url']) ?? process.env.CLOSDE_SERVER_URL ?? '';
  const authToken = stringValue(args['auth-token']) ?? process.env.CLOSDE_AUTH_TOKEN ?? '';
  const proxyUrl = stringValue(args['proxy-url']) ?? process.env.HTTPS_PROXY ?? process.env.HTTP_PROXY ?? "";
  const platform = stringValue(args.platform) ?? process.env.CLOSDE_PLATFORM ?? process.platform;
  const arch = stringValue(args.arch) ?? process.env.CLOSDE_ARCH ?? process.arch;
  const nodeVersion = stringValue(args['node-version']) ?? process.env.CLOSDE_NODE_VERSION ?? process.version;
  const claudeSettingsPath = expandConfigPath(
    stringValue(args['claude-settings']) ?? process.env.CLOSDE_CLAUDE_SETTINGS ?? DEFAULT_CLAUDE_SETTINGS,
  );
  const claudeCredentialsPath = expandConfigPath(
    stringValue(args['claude-credentials']) ?? process.env.CLOSDE_CLAUDE_CREDENTIALS ?? DEFAULT_CLAUDE_CREDENTIALS,
  );

  return {
    serverUrl,
    authToken,
    proxyUrl,
    platform,
    arch,
    nodeVersion,
    claudeSettingsPath,
    claudeCredentialsPath,
  };
}

export function parseArgs(argv: string[]): CliArgs {
  const args: CliArgs = {};

  for (let index = 0; index < argv.length; index += 1) {
    const item = argv[index];
    if (!item.startsWith('--')) {
      continue;
    }

    const [rawKey, inlineValue] = item.slice(2).split('=', 2);
    if (inlineValue !== undefined) {
      args[rawKey] = inlineValue;
      continue;
    }

    const nextItem = argv[index + 1];
    if (!nextItem || nextItem.startsWith('--')) {
      args[rawKey] = true;
      continue;
    }

    args[rawKey] = nextItem;
    index += 1;
  }

  return args;
}
