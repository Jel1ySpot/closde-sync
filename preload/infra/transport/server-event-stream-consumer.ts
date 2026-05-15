import type { ClientConfig, JsonObject } from '../../sync/types.ts';
import { ensureTrailingSlash } from '../shared/path.ts';
import { formatError, sleep } from '../shared/common.ts';
import { logger } from '../shared/logger.ts';

const SYNC_EVENT_NAME = 'config-sync';

export class ServerEventStreamConsumer {
    private reconnectDelayMs = 1000;
    private firstSyncPromise?: Promise<void>;
    private resolveFirstSync?: () => void;
    private rejectFirstSync?: (error: unknown) => void;
    private hasCompletedFirstSync = false;

    constructor(
        private readonly config: Pick<ClientConfig, 'serverUrl' | 'authToken'>,
        private readonly onSyncEvent: (event: JsonObject) => Promise<void>,
    ) { }

    async consume(): Promise<void> {
        for (; ;) {
            try {
                await this.openEventStream();
                this.reconnectDelayMs = 1000;
            } catch (error) {
                this.rejectPendingFirstSync(error);
                logger.debug(`[closde] event stream disconnected: ${formatError(error)}`);

                if (!this.hasCompletedFirstSync) {
                    throw error;
                }

                await sleep(this.reconnectDelayMs);
                this.reconnectDelayMs = Math.min(this.reconnectDelayMs * 2, 10000);
            }
        }
    }

    waitForFirstSync(): Promise<void> {
        if (this.hasCompletedFirstSync) {
            return Promise.resolve();
        }

        if (!this.firstSyncPromise) {
            this.firstSyncPromise = new Promise<void>((resolve, reject) => {
                this.resolveFirstSync = resolve;
                this.rejectFirstSync = reject;
            });
        }

        return this.firstSyncPromise;
    }

    private async openEventStream(): Promise<void> {
        const serverUrl = this.config.serverUrl;
        if (!serverUrl) {
            throw new Error('serverUrl is required to open the event stream');
        }

        const url = new URL('/events', ensureTrailingSlash(serverUrl));
        logger.debug(`[closde] event stream fetch starting: ${url.toString()}`);

        const response = await fetch(url, {
            headers: {
                Authorization: `Bearer ${this.config.authToken}`,
                Accept: 'text/event-stream',
            },
        });

        logger.debug(`[closde] event stream response: ${response.status} ${response.statusText}`);

        if (!response.ok || !response.body) {
            logger.debug(`[closde] event stream interrupted before connect: ${response.status} ${response.statusText}`);
            throw new Error(`server returned ${response.status} ${response.statusText}`);
        }

        logger.debug(`[closde] event stream connected: ${url.toString()}`);

        const decoder = new TextDecoder();
        let buffer = '';
        let currentEvent = 'message';
        let currentData = '';

        for await (const chunk of response.body) {
            const decodedChunk = decoder.decode(chunk, { stream: true });
            logger.debug(`[closde] event stream chunk received: ${decodedChunk.length} chars`);
            buffer += decodedChunk;

            while (true) {
                const newlineIndex = buffer.indexOf('\n');
                if (newlineIndex === -1) {
                    break;
                }

                const line = buffer.slice(0, newlineIndex).replace(/\r$/, '');
                buffer = buffer.slice(newlineIndex + 1);

                if (line === '') {
                    logger.debug(`[closde] event stream dispatching event=${currentEvent} dataLength=${currentData.trim().length}`);
                    await this.dispatchEvent(currentEvent, currentData);
                    currentEvent = 'message';
                    currentData = '';
                    continue;
                }

                if (line.startsWith(':')) {
                    continue;
                }

                if (line.startsWith('event:')) {
                    currentEvent = line.slice('event:'.length).trim();
                    continue;
                }

                if (line.startsWith('data:')) {
                    currentData += `${line.slice('data:'.length).trim()}\n`;
                }
            }
        }

        logger.debug(`[closde] event stream interrupted: ${url.toString()} (stream ended)`);
        throw new Error('event stream ended');
    }

    private async dispatchEvent(eventName: string, rawData: string): Promise<void> {
        const trimmedData = rawData.trim();
        logger.debug(`[closde] dispatchEvent: event=${eventName} dataLength=${trimmedData.length}`);
        if (!trimmedData || eventName !== SYNC_EVENT_NAME) {
            return;
        }

        const payload = JSON.parse(trimmedData) as unknown;
        if (!isJsonObject(payload)) {
            throw new Error('server sync payload must be a JSON object');
        }

        await this.onSyncEvent(payload);
        logger.debug('[closde] dispatchEvent: config-sync handler completed');

        if (this.hasCompletedFirstSync) {
            return;
        }

        this.hasCompletedFirstSync = true;
        logger.debug('[closde] dispatchEvent: resolving first sync promise');
        this.resolvePendingFirstSync();
    }

    private resolvePendingFirstSync(): void {
        if (!this.resolveFirstSync) {
            return;
        }

        this.resolveFirstSync();
        this.resolveFirstSync = undefined;
        this.rejectFirstSync = undefined;
    }

    private rejectPendingFirstSync(error: unknown): void {
        if (!this.rejectFirstSync) {
            return;
        }

        this.rejectFirstSync(error);
        this.resolveFirstSync = undefined;
        this.rejectFirstSync = undefined;
        this.firstSyncPromise = undefined;
    }
}

function isJsonObject(value: unknown): value is JsonObject {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
}
