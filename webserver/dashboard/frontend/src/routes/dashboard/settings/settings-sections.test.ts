import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import * as configApi from "$lib/api/objects/config/config";
import type { Config } from "$lib/api/objects/config/config";
import type { SettingsProvider } from "$lib/providers/settings/settings-provider.svelte";
import { createSettingsSections, tabs } from "./settings-sections";

vi.mock("$lib/api/objects/config/config", () => ({
    patchConfig: vi.fn(),
}));

vi.mock("$lib/utils/logger", () => ({
    log: {
        debug: vi.fn(),
        error: vi.fn(),
    },
}));

function testConfig(): Config {
    return {
        proxy: {
            listen: ":9999",
            ca_cert: "ssl/ca.crt",
            ca_key: "ssl/ca.key",
            upstream_default_https: true,
            retry_on_range_416: true,
            retry_on_invalid_range: true,
            cache_policy: {
                default_max_age: "1h",
                force_default_max_age: false,
                ignore_cache_control: true,
            },
        },
        webserver: {
            listen: ":8080",
            dashboard_disabled: false,
            api_disabled: false,
        },
        cache: {
            type: "memory",
            max_cache_size: 1024,
            cleanup_interval: "10m",
            lock_shards: 64,
            file: {
                dir: "var/cache",
            },
            memory: {
                memory_budget_percent: 25,
            },
            hybrid: {
                demote_after: "10m",
            },
        },
        logging: {
            level: "INFO",
            file: "var/log/reservoir.log",
            max_size: 1024,
            max_backups: 3,
            compress: true,
            to_stdout: true,
        },
    };
}

function testSettings(): SettingsProvider {
    return {
        dashboardSettings: {
            fields: {
                updateInterval: 10000,
            },
            reload: vi.fn(),
            save: vi.fn(),
        },
        proxySettings: {
            fields: testConfig(),
            needsRestart: false,
            reload: vi.fn(),
        },
    } as unknown as SettingsProvider;
}

function patchConfigMock() {
    return configApi.patchConfig as unknown as Mock;
}

function findSetting(settings: ReturnType<typeof createSettingsSections>, label: string) {
    const setting = Object.values(settings)
        .flat(2)
        .find((input) => input.label === label);

    if (!setting) {
        throw new Error(`Setting ${label} not found`);
    }

    return setting as {
        get: () => unknown;
        commit: (_value: unknown) => Promise<unknown>;
        label: string;
        options?: string[];
    };
}

describe("settings sections", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        patchConfigMock().mockResolvedValue("success");
    });

    it("defines the expected settings tabs and controls", () => {
        const sections = createSettingsSections(testSettings());

        expect(tabs.map((tab) => tab.id)).toEqual(["dashboard", "network", "cache", "logging"]);
        expect(
            Object.values(sections)
                .flat(2)
                .map((input) => input.label),
        ).toEqual([
            "Update Interval",
            "Proxy Listen",
            "Webserver Listen",
            "CA Certificate Path",
            "CA Key Path",
            "Upstream Default HTTPS",
            "Retry on Range 416",
            "Default Max Age",
            "Force Default Max Age",
            "Ignore Cache Control",
            "Storage Type",
            "Cache Directory",
            "Max Cache Size",
            "Cleanup Interval",
            "Memory Budget (%)",
            "Hybrid Demote After",
            "Log Level",
            "Log to Stdout",
            "Log File Path",
            "Max File Size",
        ]);
    });

    it("reads values from the current settings provider state", () => {
        const settings = testSettings();
        const sections = createSettingsSections(settings);

        expect(findSetting(sections, "Proxy Listen").get()).toBe(":9999");

        settings.proxySettings.fields.proxy.listen = ":7777";

        expect(findSetting(sections, "Proxy Listen").get()).toBe(":7777");
    });

    it("commits proxy and cache controls to the expected config paths", async () => {
        const sections = createSettingsSections(testSettings());

        await findSetting(sections, "Proxy Listen").commit(":7777");
        await findSetting(sections, "Max Cache Size").commit(4096);
        await findSetting(sections, "Hybrid Demote After").commit("5m");
        await findSetting(sections, "Ignore Cache Control").commit(false);

        expect(patchConfigMock()).toHaveBeenCalledWith("proxy.listen", ":7777");
        expect(patchConfigMock()).toHaveBeenCalledWith("cache.max_cache_size", 4096);
        expect(patchConfigMock()).toHaveBeenCalledWith("cache.hybrid.demote_after", "5m");
        expect(patchConfigMock()).toHaveBeenCalledWith(
            "proxy.cache_policy.ignore_cache_control",
            false,
        );
    });

    it("offers the hybrid cache backend in storage type options", () => {
        const storageType = findSetting(createSettingsSections(testSettings()), "Storage Type");

        expect(storageType.options).toEqual(["memory", "file", "hybrid"]);
    });

    it("marks proxy settings as requiring restart when config patch reports it", async () => {
        const settings = testSettings();
        patchConfigMock().mockResolvedValueOnce("restart required");

        await findSetting(createSettingsSections(settings), "Storage Type").commit("file");

        expect(settings.proxySettings.needsRestart).toBe(true);
    });

    it("updates and saves dashboard-local settings", async () => {
        const settings = testSettings();

        await findSetting(createSettingsSections(settings), "Update Interval").commit(5000);

        expect(settings.dashboardSettings.fields.updateInterval).toBe(5000);
        expect(settings.dashboardSettings.save).toHaveBeenCalled();
        expect(patchConfigMock()).not.toHaveBeenCalled();
    });
});
