export type JsonObject = Record<string, unknown>;

export type ClientConfig = {
  serverUrl?: string;
  authToken?: string;
  proxyUrl?: string;
  platform: string;
  arch: string;
  nodeVersion: string;
  claudeSettingsPath: string;
  claudeCredentialsPath: string;
};
