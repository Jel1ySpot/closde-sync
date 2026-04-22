import { homedir } from 'node:os';
import { resolve } from 'node:path';

export function ensureTrailingSlash(url: string): string {
  return url.endsWith('/') ? url : `${url}/`;
}

export function expandConfigPath(pathValue: string): string {
  const trimmed = pathValue.trim();
  const home = homedir();

  if (trimmed === '$HOME' || trimmed === '${HOME}' || trimmed === '~') {
    return home;
  }

  if (trimmed.startsWith('$HOME/')) {
    return resolve(home, trimmed.slice('$HOME/'.length));
  }

  if (trimmed.startsWith('${HOME}/')) {
    return resolve(home, trimmed.slice('${HOME}/'.length));
  }

  if (trimmed.startsWith('~/')) {
    return resolve(home, trimmed.slice(2));
  }

  return resolve(trimmed);
}
