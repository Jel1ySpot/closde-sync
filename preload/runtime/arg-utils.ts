export function stringValue(value: string | boolean | undefined): string | undefined {
    return typeof value === 'string' ? value : undefined;
}
