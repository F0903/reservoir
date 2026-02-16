import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from "vitest";
import { MetricsProvider } from "./metrics-provider.svelte";
import * as metricsApi from "$lib/api/objects/metrics/metrics";
import type { SettingsProvider } from "../settings/settings-provider.svelte";

// Mock Svelte/Kit environment
vi.mock("$app/environment", () => ({
    browser: true,
}));

// Mock the API
vi.mock("$lib/api/objects/metrics/metrics", () => ({
    getAllMetrics: vi.fn(),
}));

// Mock logger to keep test output clean
vi.mock("$lib/utils/logger", () => ({
    log: {
        debug: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
    },
}));

describe("MetricsProvider", () => {
    let settings: SettingsProvider;
    let provider: MetricsProvider;
    let mockMetrics: metricsApi.Metrics;

    beforeEach(() => {
        vi.useFakeTimers();

        mockMetrics = {
            cache: {
                cache_hits: 10,
                cache_misses: 2,
                cache_errors: 0,
                cache_entries: 5,
                bytes_cached: 1024,
                cleanup_runs: 1,
                bytes_cleaned: 0,
                cache_evictions: 0,
                cache_hit_latency: 5,
                cache_miss_latency: 100,
            },
            requests: {
                http_proxy_requests: 8,
                https_proxy_requests: 4,
                bytes_served: 1000,
                bytes_fetched: 800,
                upstream_requests: 6,
                client_request_latency: 50,
                upstream_request_latency: 40,
                coalesced_requests: 2,
                non_coalesced_requests: 10,
                coalesced_cache_hits: 1,
                coalesced_cache_revalidations: 0,
                coalesced_cache_misses: 1,
                status_ok_responses: 10,
                status_client_error_responses: 2,
                status_server_error_responses: 0,
            },
            system: {
                num_goroutines: 5,
                mem_alloc_bytes: 1024,
                mem_total_alloc_bytes: 2048,
                mem_sys_bytes: 4096,
                start_time: new Date().toISOString(),
            },
        };

        // Minimal mock of SettingsProvider
        settings = {
            dashboardSettings: {
                fields: {
                    updateInterval: 1000,
                },
                reload: () => Promise.resolve(),
                save: () => {},
            },
            // We are only gonna be using dashboardSettings, so this is fine
        } as unknown as SettingsProvider;

        (metricsApi.getAllMetrics as Mock).mockResolvedValue(mockMetrics);
        provider = new MetricsProvider(settings);
    });

    afterEach(() => {
        provider.stopRefresh();
        vi.clearAllMocks();
        vi.clearAllTimers();
        vi.useRealTimers();
    });

    it("should initialize with default state", () => {
        expect(provider.data).toBeNull();
        expect(provider.error).toBeNull();
        expect(provider.loading).toBe(false);
        expect(provider.lastUpdated).toBeNull();
    });

    it("should fetch metrics and update state", async () => {
        await provider.refreshMetrics();

        expect(provider.data).toEqual(mockMetrics);
        expect(provider.loading).toBe(false);
        expect(provider.error).toBeNull();
        expect(provider.lastUpdated).toBeInstanceOf(Date);
    });

    it("should handle fetch errors", async () => {
        (metricsApi.getAllMetrics as Mock).mockRejectedValue(new Error("Network Error"));

        await provider.refreshMetrics();

        expect(provider.error).toBe("Network Error");
        expect(provider.loading).toBe(false);
        expect(provider.data).toBeNull();
    });

    it("should toggle loading state during fetch", async () => {
        let resolveFetch: (_value: metricsApi.Metrics) => void = () => {};
        const promise = new Promise<metricsApi.Metrics>((resolve) => (resolveFetch = resolve));
        (metricsApi.getAllMetrics as Mock).mockReturnValue(promise);

        const refreshPromise = provider.refreshMetrics();

        expect(provider.loading).toBe(true);

        resolveFetch(mockMetrics);
        await refreshPromise;

        expect(provider.loading).toBe(false);
    });

    it("should run the refresh loop", async () => {
        provider.startRefresh();

        // Should have called refresh once immediately (via the loop start)
        expect(metricsApi.getAllMetrics).toHaveBeenCalledTimes(1);

        // Advance time to trigger next refresh
        await vi.advanceTimersByTimeAsync(1000);
        expect(metricsApi.getAllMetrics).toHaveBeenCalledTimes(2);

        await vi.advanceTimersByTimeAsync(1000);
        expect(metricsApi.getAllMetrics).toHaveBeenCalledTimes(3);
    });

    it("should stop the refresh loop", async () => {
        provider.startRefresh();
        expect(metricsApi.getAllMetrics).toHaveBeenCalledTimes(1);

        provider.stopRefresh();

        await vi.advanceTimersByTimeAsync(2000);
        // Should not have increased
        expect(metricsApi.getAllMetrics).toHaveBeenCalledTimes(1);
    });

    it("should adapt to interval changes in settings", async () => {
        provider.startRefresh();
        expect(metricsApi.getAllMetrics).toHaveBeenCalledTimes(1);

        // Change interval to 5 seconds
        settings.dashboardSettings.fields.updateInterval = 5000;

        await vi.advanceTimersByTimeAsync(1000);
        // Should NOT have triggered yet
        expect(metricsApi.getAllMetrics).toHaveBeenCalledTimes(1);

        await vi.advanceTimersByTimeAsync(4000);
        // Should trigger now (total 5000ms)
        expect(metricsApi.getAllMetrics).toHaveBeenCalledTimes(2);
    });
});
