import Module from 'node:module';
import { installProxyInjection } from './inject/proxy.ts';
import { installProcessInjection } from './inject/process.ts';
import { HELP_TEXT, loadConfig, parseArgs } from './runtime/cli.ts';
import { ClosdeClient } from './runtime/closde-client.ts';
import { formatError } from './infra/shared/common.ts';
import { logger } from './infra/shared/logger.ts';

async function main(): Promise<void> {
  const args = parseArgs(process.argv.slice(2));

  if (args.help) {
    process.stdout.write(HELP_TEXT);
    return;
  }

  try {
    logger.info('[closde] preload starting');

    const config = loadConfig(args);
    installProcessInjection({
      platform: config.platform,
      arch: config.arch,
      nodeVersion: config.nodeVersion,
    });

    if (config.proxyUrl) {
      installProxyInjection(config.proxyUrl);
    }

    if (config.serverUrl) {
      const client = new ClosdeClient(config);
      await client.start();
    }
  } catch (error) {
    handleFatal(error);
  }
}

const originalRunMain = Module.runMain;
let preloadStarted = false;

Module.runMain = function runMainWithClosdePreload(...args: Parameters<typeof originalRunMain>) {
  if (preloadStarted) {
    return originalRunMain.apply(this, args);
  }

  preloadStarted = true;
  void main().then(() => {
    originalRunMain.apply(this, args);
  });
};

function handleFatal(error: unknown): never {
  logger.debug(`[closde] fatal: ${formatError(error)}`);
  process.exit(1);
}
