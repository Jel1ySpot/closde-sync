import log from 'loglevel';

const DEFAULT_LEVEL = resolveLogLevel(process.env.DEBUG_MODE);

log.setDefaultLevel(DEFAULT_LEVEL);
log.setLevel(DEFAULT_LEVEL);

export const logger = {
    info(message: string): void {
        log.info(message);
    },
    debug(message: string): void {
        log.debug(message);
    },
    error(message: string): void {
        log.error(message);
    },
};

function resolveLogLevel(value: string | undefined): log.LogLevelDesc {
    return isDebugModeEnabled(value) ? 'debug' : 'info';
}

function isDebugModeEnabled(value: string | undefined): boolean {
    switch ((value ?? '').trim().toLowerCase()) {
        case '1':
        case 'true':
        case 'yes':
        case 'on':
        case 'debug':
            return true;
        default:
            return false;
    }
}
