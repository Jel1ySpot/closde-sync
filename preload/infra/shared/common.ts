import { createHash } from 'node:crypto';

export function sha256(input: Uint8Array): string {
    return createHash('sha256').update(input).digest('hex');
}

export function stableJsonStringify(value: unknown): string {
    return JSON.stringify(sortJsonValue(value));
}

export function formatError(error: unknown): string {
    return error instanceof Error ? error.message : String(error);
}

export function sleep(milliseconds: number): Promise<void> {
    return new Promise((resolvePromise) => {
        setTimeout(resolvePromise, milliseconds);
    });
}

function sortJsonValue(value: unknown): unknown {
    if (Array.isArray(value)) {
        return value.map(sortJsonValue);
    }

    if (value && typeof value === 'object') {
        return Object.fromEntries(
            Object.entries(value as Record<string, unknown>)
                .sort(([left], [right]) => left.localeCompare(right))
                .map(([key, nestedValue]) => [key, sortJsonValue(nestedValue)]),
        );
    }

    return value;
}
