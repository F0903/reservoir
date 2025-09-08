import {
    apiGet,
    type FetchFn,
    type APIObjectConstructor,
    APIJsonObject,
    apiPatch,
} from "$lib/api/api-object";
import { setPropIfChanged } from "$lib/utils/objects";

export class Config {
    configVersion: number = $state(0);
    proxyListen: string = $state("");
    caCert: string = $state("");
    caKey: string = $state("");
    webserverListen: string = $state("");
    dashboardDisabled: boolean = $state(false);
    apiDisabled: boolean = $state(false);
    cacheDir: string = $state("");
    ignoreCacheControl: boolean = $state(false);
    maxCacheSize: number = $state(0);
    defaultCacheMaxAge: string = $state(""); // Duration formatted string
    forceDefaultCacheMaxAge: boolean = $state(false);
    cacheCleanupInterval: string = $state(""); // Duration formatted string
    upstreamDefaultHttps: boolean = $state(false);
    logLevel: string = $state("");
    logFile: string = $state("");
    logFileMaxSize: number = $state(0);
    logFileMaxBackups: number = $state(0);
    logFileCompress: boolean = $state(false);
    logToStdout: boolean = $state(false);

    constructor(json: Record<string, unknown>) {
        this.updateFrom(json);
    }

    // prettier-ignore
    updateFrom = (json: Record<string, unknown>) => {
        setPropIfChanged("config_version",              json, this.configVersion,           (value) => this.configVersion = value as number);
        setPropIfChanged("proxy_listen",                json, this.proxyListen,             (value) => this.proxyListen = value as string);
        setPropIfChanged("ca_cert",                     json, this.caCert,                  (value) => this.caCert = value as string);
        setPropIfChanged("ca_key",                      json, this.caKey,                   (value) => this.caKey = value as string);
        setPropIfChanged("webserver_listen",            json, this.webserverListen,         (value) => this.webserverListen = value as string);
        setPropIfChanged("dashboard_disabled",          json, this.dashboardDisabled,       (value) => this.dashboardDisabled = value as boolean);
        setPropIfChanged("api_disabled",                json, this.apiDisabled,             (value) => this.apiDisabled = value as boolean);
        setPropIfChanged("cache_dir",                   json, this.cacheDir,                (value) => this.cacheDir = value as string);
        setPropIfChanged("ignore_cache_control",        json, this.ignoreCacheControl,      (value) => this.ignoreCacheControl = value as boolean);
        setPropIfChanged("max_cache_size",              json, this.maxCacheSize,            (value) => this.maxCacheSize = value as number);
        setPropIfChanged("default_cache_max_age",       json, this.defaultCacheMaxAge,      (value) => this.defaultCacheMaxAge = value as string);
        setPropIfChanged("force_default_cache_max_age", json, this.forceDefaultCacheMaxAge, (value) => this.forceDefaultCacheMaxAge = value as boolean);
        setPropIfChanged("cache_cleanup_interval",      json, this.cacheCleanupInterval,    (value) => this.cacheCleanupInterval = value as string);
        setPropIfChanged("upstream_default_https",      json, this.upstreamDefaultHttps,    (value) => this.upstreamDefaultHttps = value as boolean);
        setPropIfChanged("log_level",                   json, this.logLevel,                (value) => this.logLevel = value as string);
        setPropIfChanged("log_file",                    json, this.logFile,                 (value) => this.logFile = value as string);
        setPropIfChanged("log_file_max_size",           json, this.logFileMaxSize,          (value) => this.logFileMaxSize = value as number);
        setPropIfChanged("log_file_max_backups",        json, this.logFileMaxBackups,       (value) => this.logFileMaxBackups = value as number);
        setPropIfChanged("log_file_compress",           json, this.logFileCompress,         (value) => this.logFileCompress = value as boolean);
        setPropIfChanged("log_to_stdout",               json, this.logToStdout,             (value) => this.logToStdout = value as boolean);
    }

    // Updates the config object by fetching from the API
    update = async () => {
        const data = await getConfig(APIJsonObject);
        this.updateFrom(data as Record<string, unknown>);
    };
}

export async function getConfig<C extends APIObjectConstructor<T>, T>(
    type: C = Config as C,
    fetchFn: FetchFn = fetch,
): Promise<T> {
    return apiGet<T>("/config", type, fetchFn);
}

export async function patchConfig(
    propName: string,
    value: unknown,
    fetchFn: FetchFn = fetch,
): Promise<void> {
    return apiPatch("/config", { [propName]: value }, fetchFn);
}
