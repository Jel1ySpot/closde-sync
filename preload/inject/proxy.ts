import { ProxyAgent } from 'proxy-agent';
import { ProxyAgent as UndiciProxyAgent, setGlobalDispatcher } from 'undici';
import http from 'node:http';
import https from 'node:https';
import { syncBuiltinESMExports } from 'node:module';

type RequestFn = (...args: any[]) => http.ClientRequest;
type PatchableModule = {
  request: RequestFn;
  get: RequestFn;
};

export function installProxyInjection(proxyUrl?: string): void {
  if (!proxyUrl || process.env.__HTTP_PATCH_PROXY_ACTIVE === '1') {
    return;
  }

  const agent = new ProxyAgent({
    getProxyForUrl: () => proxyUrl,
  });
  setGlobalDispatcher(new UndiciProxyAgent(proxyUrl));

  const mutableHttp = http as typeof http & PatchableModule;
  const mutableHttps = https as typeof https & PatchableModule;

  const originalHttpRequest = mutableHttp.request;
  const originalHttpsRequest = mutableHttps.request;

  mutableHttp.request = patchRequest(mutableHttp, originalHttpRequest, agent);
  mutableHttps.request = patchRequest(mutableHttps, originalHttpsRequest, agent);

  mutableHttp.get = function patchedGet(...args: any[]) {
    const req = mutableHttp.request(...args);
    req.end();
    return req;
  };

  mutableHttps.get = function patchedGet(...args: any[]) {
    const req = mutableHttps.request(...args);
    req.end();
    return req;
  };

  syncBuiltinESMExports();

  process.env.__HTTP_PATCH_PROXY_ACTIVE = '1';
}

function patchRequest(
  mod: PatchableModule,
  originalRequest: RequestFn,
  agent: ProxyAgent
): RequestFn {
  return function patchedRequest(...args: any[]) {
    let urlOrOptions = args[0];
    let options;
    let callback;

    if (typeof urlOrOptions === 'string' || urlOrOptions instanceof URL) {
      options = args[1];
      callback = args[2];

      options = options == null ? {} : { ...options };

      if (options.agent == null) {
        options.agent = agent;
      }

      return originalRequest.call(mod, urlOrOptions, options, callback);
    }

    options = urlOrOptions ? { ...urlOrOptions } : {};
    callback = args[1];

    if (options.agent == null) {
      options.agent = agent;
    }

    return originalRequest.call(mod, options, callback);
  };
}