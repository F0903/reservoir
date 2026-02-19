import { apiGet, type FetchFn, apiPatch } from "$lib/api/api-helpers";
import type { DeepPartial } from "$lib/utils/patch";

export type CachePolicyConfig = {
    ignore_cache_control: boolean;
    default_max_age: string;
    force_default_max_age: boolean;
};

export type ProxyConfig = {
    listen: string;
    ca_cert: string;
    ca_key: string;
    upstream_default_https: boolean;
    retry_on_range_416: boolean;
    retry_on_invalid_range: boolean;
    cache_policy: CachePolicyConfig;
};

export type WebserverConfig = {
    listen: string;
    dashboard_disabled: boolean;
    api_disabled: boolean;
};

export type FileCacheConfig = {
    dir: string;
};

export type MemoryCacheConfig = {
    memory_budget_percent: number;
};

export type CacheConfig = {
    max_cache_size: number;
    type: string;
    cleanup_interval: string;
    lock_shards: number;
    file: FileCacheConfig;
    memory: MemoryCacheConfig;
};

export type LogConfig = {
    level: string;
    file: string;
    max_size: number;
    max_backups: number;
    compress: boolean;
    to_stdout: boolean;
};

export type Config = {
    proxy: ProxyConfig;
    webserver: WebserverConfig;
    cache: CacheConfig;
    logging: LogConfig;
};

type Leaves<T, P extends string = ""> = T extends object
    ? { [K in keyof T]: K extends string ? Leaves<T[K], `${P}${K}.`> : never }[keyof T]
    : P extends `${infer S}.`
      ? S
      : never;

export type ConfigPropPath = Leaves<Config>;

export async function getConfig(fetchFn: FetchFn = fetch): Promise<Readonly<Config>> {
    return apiGet<Config>("/config", fetchFn);
}

export async function patchConfig(
    keyPath: ConfigPropPath,
    value: unknown,
    fetchFn: FetchFn = fetch,
): Promise<string> {
    // Create nested object from keyPath (e.g. "proxy.listen" -> { proxy: { listen: value } })
    const parts = keyPath.split(".");
    const body: Record<string, unknown> = {};
    let current = body;

    for (let i = 0; i < parts.length - 1; i++) {
        const part = parts[i];
        if (!(part in current)) {
            current[part] = {};
        }
        current = current[part] as Record<string, unknown>;
    }
    current[parts[parts.length - 1]] = value;

    return apiPatch("/config", body as DeepPartial<Config>, fetchFn);
}
