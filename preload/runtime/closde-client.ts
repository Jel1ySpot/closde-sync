import type { ClientConfig } from '../sync/types.ts';
import { FixedFileWatcher } from '../infra/watch/fixed-file-watcher.ts';
import { FileSyncService } from '../sync/file-sync-service.ts';
import { ServerEventStreamConsumer } from '../infra/transport/server-event-stream-consumer.ts';
import { logger } from '../infra/shared/logger.ts';

export class ClosdeClient {
  private readonly fileSync: FileSyncService;
  private readonly settingsWatcher: FixedFileWatcher;
  private readonly credentialsWatcher: FixedFileWatcher;
  private readonly serverEvents: ServerEventStreamConsumer;
  private hasReceivedInitialRemoteState = false;

  constructor(private readonly config: ClientConfig) {
    this.fileSync = new FileSyncService(config);
    const handleLocalChange = async (): Promise<void> => {
      if (this.fileSync.isApplyingRemoteState()) {
        logger.debug('[closde] ignored local file change while applying remote state');
        return;
      }
      logger.debug('[closde] local file change detected, uploading merged state');
      await this.fileSync.uploadState();
    };

    this.settingsWatcher = new FixedFileWatcher(config.claudeSettingsPath, handleLocalChange);
    this.credentialsWatcher = new FixedFileWatcher(config.claudeCredentialsPath, handleLocalChange);
    this.serverEvents = new ServerEventStreamConsumer(config, async (state) => {
      const phase = this.hasReceivedInitialRemoteState ? 'remote state update' : 'initial remote state';
      logger.debug(`[closde] received ${phase} from ${this.config.serverUrl}`);
      await this.fileSync.applyRemoteState(state);
      if (!this.hasReceivedInitialRemoteState) {
        this.hasReceivedInitialRemoteState = true;
        logger.debug('[closde] initial remote state applied to local files');
      }
    });
  }

  async start(): Promise<void> {
    logger.debug('[closde] initializing local sync state');
    await this.fileSync.initializeLocalState();

    logger.debug('[closde] starting local file watchers');
    await Promise.all([this.settingsWatcher.start(), this.credentialsWatcher.start()]);

    logger.debug(`[closde] opening event stream to ${this.config.serverUrl}`);
    const consumePromise = this.serverEvents.consume();
    const firstSyncPromise = this.serverEvents.waitForFirstSync();
    void consumePromise.catch((error) => {
      logger.debug(`[closde] event stream stopped: ${error instanceof Error ? error.message : String(error)}`);
    });

    logger.debug('[closde] waiting for initial remote state before continuing');
    await firstSyncPromise;

    logger.debug(`[closde] watching settings=${this.config.claudeSettingsPath}`);
    logger.debug(`[closde] watching credentials=${this.config.claudeCredentialsPath}`);
    logger.info(`[closde] connected to ${this.config.serverUrl}`);
  }
}
