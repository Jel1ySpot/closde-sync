import type { ClientConfig, JsonObject } from './types.ts';
import { readJsonObject, writeJsonObject } from '../infra/fs/json-file-store.ts';
import { ensureTrailingSlash } from '../infra/shared/path.ts';
import { stableJsonStringify } from '../infra/shared/common.ts';
import { logger } from '../infra/shared/logger.ts';

const CLAUDE_SETTINGS_SYNC_KEYS = [
    'userID',
    'firstStartTime',
    'oauthAccount',
    'claudeCodeFirstTokenDate',
    'groveConfigCache',
    'passesEligibilityCache',
    'overageCreditGrantCache',
    'claudeAiMcpEverConnected',
] as const;

const CREDENTIALS_KEY = 'credentials';

export class FileSyncService {
    private syncedState: JsonObject = {};
    private applyingRemoteState = false;

    constructor(private readonly config: ClientConfig) { }

    async initializeLocalState(): Promise<void> {
        logger.debug('[closde] initializeLocalState: building merged state');
        this.syncedState = await this.buildMergedState();
        logger.debug('[closde] initializeLocalState: merged state ready');
    }

    isApplyingRemoteState(): boolean {
        return this.applyingRemoteState;
    }

    async applyRemoteState(nextState: JsonObject): Promise<void> {
        if (sameJsonValue(this.syncedState, nextState)) {
            return;
        }

        this.applyingRemoteState = true;
        try {
            const normalizedState = normalizeIncomingState(cloneJsonObject(nextState));
            await this.writeMergedState(normalizedState);
            this.syncedState = normalizedState;
            logger.debug('[closde] applied remote state update');
        } finally {
            this.applyingRemoteState = false;
        }
    }

    async uploadState(): Promise<void> {
        const serverUrl = this.config.serverUrl;
        if (!serverUrl) {
            throw new Error('serverUrl is required to upload sync state');
        }

        const currentState = await this.buildMergedState();
        if (sameJsonValue(this.syncedState, currentState)) {
            return;
        }

        const response = await fetch(new URL('/sync', ensureTrailingSlash(serverUrl)), {
            method: 'POST',
            headers: {
                Authorization: `Bearer ${this.config.authToken}`,
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(currentState),
        });

        if (!response.ok) {
            throw new Error(`upload failed: ${response.status} ${response.statusText}`);
        }

        this.syncedState = cloneJsonObject(currentState);
        logger.debug('[closde] uploaded merged state');
    }

    private async buildMergedState(): Promise<JsonObject> {
        logger.debug(`[closde] buildMergedState: reading settings from ${this.config.claudeSettingsPath}`);
        const settingsPromise = readJsonObject(this.config.claudeSettingsPath).then((value) => {
            logger.debug(`[closde] buildMergedState: finished settings read from ${this.config.claudeSettingsPath}`);
            return value;
        });
        logger.debug(`[closde] buildMergedState: reading credentials from ${this.config.claudeCredentialsPath}`);
        const credentialsPromise = readJsonObject(this.config.claudeCredentialsPath).then((value) => {
            logger.debug(`[closde] buildMergedState: finished credentials read from ${this.config.claudeCredentialsPath}`);
            return value;
        });
        const [settings, credentials] = await Promise.all([settingsPromise, credentialsPromise]);

        const mergedState: JsonObject = {};
        for (const key of CLAUDE_SETTINGS_SYNC_KEYS) {
            if (Object.prototype.hasOwnProperty.call(settings, key)) {
                mergedState[key] = cloneJsonValue(settings[key]);
            }
        }

        if (Object.keys(credentials).length > 0) {
            mergedState[CREDENTIALS_KEY] = cloneJsonObject(credentials);
        }

        return mergedState;
    }

    private async writeMergedState(nextState: JsonObject): Promise<void> {
        const [currentSettings, currentCredentials] = await Promise.all([
            readJsonObject(this.config.claudeSettingsPath),
            readJsonObject(this.config.claudeCredentialsPath),
        ]);

        const nextSettings = cloneJsonObject(currentSettings);
        for (const key of CLAUDE_SETTINGS_SYNC_KEYS) {
            if (Object.prototype.hasOwnProperty.call(nextState, key)) {
                nextSettings[key] = cloneJsonValue(nextState[key]);
                continue;
            }
            delete nextSettings[key];
        }

        const rawCredentials = nextState[CREDENTIALS_KEY];
        const nextCredentials = isJsonObject(rawCredentials) ? cloneJsonObject(rawCredentials) : {};

        await Promise.all([
            writeJsonObject(this.config.claudeSettingsPath, nextSettings),
            writeJsonObject(this.config.claudeCredentialsPath, mergeCredentialsFallback(currentCredentials, nextCredentials)),
        ]);
    }
}

function normalizeIncomingState(nextState: JsonObject): JsonObject {
    const normalized: JsonObject = {};

    for (const key of CLAUDE_SETTINGS_SYNC_KEYS) {
        if (Object.prototype.hasOwnProperty.call(nextState, key)) {
            normalized[key] = cloneJsonValue(nextState[key]);
        }
    }

    const rawCredentials = nextState[CREDENTIALS_KEY];
    if (isJsonObject(rawCredentials)) {
        normalized[CREDENTIALS_KEY] = cloneJsonObject(rawCredentials);
    }

    return normalized;
}

function mergeCredentialsFallback(_currentCredentials: JsonObject, nextCredentials: JsonObject): JsonObject {
    return nextCredentials;
}

function sameJsonValue(left: unknown, right: unknown): boolean {
    return stableJsonStringify(left) === stableJsonStringify(right);
}

function isJsonObject(value: unknown): value is JsonObject {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function cloneJsonObject(value: JsonObject): JsonObject {
    return JSON.parse(JSON.stringify(value)) as JsonObject;
}

function cloneJsonValue(value: unknown): unknown {
    return JSON.parse(JSON.stringify(value ?? null));
}
