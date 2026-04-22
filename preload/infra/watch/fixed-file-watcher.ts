import { type Stats, watchFile } from 'node:fs';

import type { FileChangeEvent } from './types.ts';
import { formatError } from '../shared/common.ts';
import { logger } from '../shared/logger.ts';

export class FixedFileWatcher {
  private pendingTimer?: NodeJS.Timeout;
  private started = false;

  constructor(
    private readonly absolutePath: string,
    private readonly onFileChange: (change: FileChangeEvent) => Promise<void>,
  ) {}

  async start(): Promise<void> {
    if (this.started) {
      return;
    }

    try {
      logger.debug(`[closde] starting watching file ${this.absolutePath}`);
      watchFile(
        this.absolutePath,
        { persistent: true, interval: 150 },
        (current, previous) => {
          const eventType = detectEventType(current, previous);
          if (!eventType) {
            return;
          }

          this.scheduleChange(eventType);
        },
      );
      this.started = true;
    } catch (error) {
      logger.debug(`[closde] watcher error on ${this.absolutePath}: ${formatError(error)}`);
      throw error;
    }
  }

  private scheduleChange(eventType: FileChangeEvent['eventType']): void {
    if (this.pendingTimer) {
      clearTimeout(this.pendingTimer);
    }

    this.pendingTimer = setTimeout(() => {
      void this.onFileChange({
        absolutePath: this.absolutePath,
        eventType,
      });
    }, 150);
  }
}

function detectEventType(current: Stats, previous: Stats): FileChangeEvent['eventType'] | undefined {
  const currentExists = current.nlink > 0;
  const previousExists = previous.nlink > 0;

  if (currentExists !== previousExists) {
    return 'rename';
  }

  if (!currentExists) {
    return undefined;
  }

  if (current.ino !== previous.ino) {
    return 'rename';
  }

  if (current.mtimeMs !== previous.mtimeMs || current.size !== previous.size || current.mode !== previous.mode) {
    return 'change';
  }

  return undefined;
}
