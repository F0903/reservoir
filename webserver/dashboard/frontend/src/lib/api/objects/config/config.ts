import { apiGet, type FetchFn, apiPatch } from "$lib/api/api-helpers";

export type Config = {
    config_version: number;
    proxy_listen: string;
    ca_cert: string;
    ca_key: string;
    webserver_listen: string;
    dashboard_disabled: boolean;
    api_disabled: boolean;
    ignore_cache_control: boolean;
    max_cache_size: number;
    default_cache_max_age: string;
    force_default_cache_max_age: boolean;
    cache_dir: string;
    cache_cleanup_interval: string;
    cache_lock_shards: number;
    upstream_default_https: boolean;
    retry_on_range_416: boolean;
    retry_on_invalid_range: boolean;
    log_level: string;
    log_file: string;
    log_file_max_size: number;
    log_file_max_backups: number;
    log_file_compress: boolean;
    log_to_stdout: boolean;
};

export async function getConfig(fetchFn: FetchFn = fetch): Promise<Readonly<Config>> {
    return apiGet<Config>("/config", fetchFn);
}

export async function patchConfig(
    propName: string,
    value: unknown,
    fetchFn: FetchFn = fetch,
): Promise<string> {
    return apiPatch("/config", { [propName]: value }, fetchFn);
}
