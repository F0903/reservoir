import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import * as configApi from "#lib/api/objects/config/config.ts";
import type { Config } from "#lib/api/objects/config/config.ts";
import type { SettingsProvider } from "#lib/providers/settings/settings-provider.svelte.ts";
import { createSettingsSections } from "./settings-sections";

vi.mock("#lib/api/objects/config/config.ts", () => ({
    patchConfig: vi.fn(),
}));

vi.mock("#lib/utils/logger.ts", () => ({
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
            upgrade_http_to_https: true,
            retry_on_range_416: true,
            retry_on_invalid_range: true,
            cache_policy: {
                default_max_age: "15m",
                force_default_max_age: true,
                ignore_cache_control: true,
            },
        },
        webserver: {
            listen: ":8080",
            dashboard_disabled: false,
            api_disabled: false,
            trusted_proxies: "",
        },
        cache: {
            type: "hybrid",
            max_cache_size: "10G",
            cleanup_interval: "5m",
            lock_shards: 64,
            file: {
                dir: "var/cache",
            },
            memory: {
                memory_budget_percent: 25,
            },
            hybrid: {
                demote_after: "5m",
            },
        },
        logging: {
            level: "INFO",
            file: "var/log/reservoir.log",
            max_size: "500M",
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
    };
}

describe("settings sections", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        patchConfigMock().mockResolvedValue("success");
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
        await findSetting(sections, "Total Cache Limit").commit("4G");
        await findSetting(sections, "Demote Idle Memory After").commit("5m");
        await findSetting(sections, "Ignore Cache Control").commit(false);

        expect(patchConfigMock()).toHaveBeenCalledWith("proxy.listen", ":7777");
        expect(patchConfigMock()).toHaveBeenCalledWith("cache.max_cache_size", "4G");
        expect(patchConfigMock()).toHaveBeenCalledWith("cache.hybrid.demote_after", "5m");
        expect(patchConfigMock()).toHaveBeenCalledWith(
            "proxy.cache_policy.ignore_cache_control",
            false,
        );
    });

    it("marks proxy settings as requiring restart when config patch reports it", async () => {
        const settings = testSettings();
        patchConfigMock().mockResolvedValueOnce("restart required");

        await findSetting(createSettingsSections(settings), "Cache Backend").commit("file");

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
