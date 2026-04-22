import { readFileSync } from 'node:fs';
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname } from 'node:path';

import { logger } from '../shared/logger.ts';

export type JsonObject = Record<string, unknown>;

export async function readJsonObject(filePath: string): Promise<JsonObject> {
  logger.debug(`[closde] readJsonObject: start ${filePath}`);
  try {
    const content = readFileSync(filePath, 'utf8');
    logger.debug(`[closde] readJsonObject: read ${content.length} bytes from ${filePath}`);
    if (!content.trim()) {
      logger.debug(`[closde] readJsonObject: empty JSON object for ${filePath}`);
      return {};
    }

    logger.debug(`[closde] readJsonObject: parsing JSON from ${filePath}`);
    const parsed = JSON.parse(content) as unknown;
    if (!isJsonObject(parsed)) {
      throw new Error(`${filePath} must contain a JSON object`);
    }

    logger.debug(`[closde] readJsonObject: parsed JSON object from ${filePath}`);
    return parsed;
  } catch (error) {
    if (isFileNotFound(error)) {
      logger.debug(`[closde] readJsonObject: file not found ${filePath}, using empty object`);
      return {};
    }

    logger.debug(`[closde] readJsonObject: failed for ${filePath}: ${error instanceof Error ? error.message : String(error)}`);
    throw error;
  }
}

export async function writeJsonObject(filePath: string, value: JsonObject): Promise<void> {
  await mkdir(dirname(filePath), { recursive: true });
  await writeFile(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

function isJsonObject(value: unknown): value is JsonObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isFileNotFound(error: unknown): error is NodeJS.ErrnoException {
  return typeof error === 'object' && error !== null && 'code' in error && error.code === 'ENOENT';
}
