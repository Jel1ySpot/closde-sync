export type ProcessInjectionConfig = {
  platform?: string;
  arch?: string;
  nodeVersion?: string;
};

export function installProcessInjection(config: ProcessInjectionConfig): void {
  overrideReadonlyProperty(process, 'platform', config.platform);
  overrideReadonlyProperty(process, 'arch', config.arch);
  overrideReadonlyProperty(process, 'version', config.nodeVersion);
}

function overrideReadonlyProperty<T extends object>(target: T, key: keyof T, value: T[keyof T] | undefined): void {
  if (value == null || value === '') {
    return;
  }

  Object.defineProperty(target, key, {
    configurable: true,
    enumerable: true,
    value,
  });
}
