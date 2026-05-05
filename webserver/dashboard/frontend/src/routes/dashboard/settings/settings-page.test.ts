import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import * as configApi from "$lib/api/objects/config/config";
import type { Config } from "$lib/api/objects/config/config";
import * as restartRequiredApi from "$lib/api/objects/config/restart-required";
import { SettingsProvider } from "$lib/providers/settings/settings-provider.svelte";
import SettingsPage from "./+page.svelte";

const contextMocks = vi.hoisted(() => ({
    settings: undefined as SettingsProvider | undefined,
    toast: {
        success: vi.fn(),
        error: vi.fn(),
    },
}));

vi.mock("$app/environment", () => ({
    browser: true,
}));

vi.mock("$lib/api/objects/config/config", () => ({
    getConfig: vi.fn(),
    patchConfig: vi.fn(),
}));

vi.mock("$lib/api/objects/config/restart-required", () => ({
    getRestartRequired: vi.fn(),
}));

vi.mock("$lib/context", () => ({
    getSettingsProvider: () => contextMocks.settings,
    getToastProvider: () => contextMocks.toast,
}));

vi.mock("$lib/utils/logger", () => ({
    log: {
        debug: vi.fn(),
        error: vi.fn(),
    },
}));

function baseConfig(): Config {
    return {
        proxy: {
            listen: ":9999",
            ca_cert: "ssl/ca.crt",
            ca_key: "ssl/ca.key",
            upstream_default_https: true,
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
        },
        cache: {
            type: "hybrid",
            max_cache_size: 1024,
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
            max_size: 1024,
            max_backups: 3,
            compress: true,
            to_stdout: true,
        },
    };
}

function getConfigMock() {
    return configApi.getConfig as unknown as Mock;
}

function patchConfigMock() {
    return configApi.patchConfig as unknown as Mock;
}

function getRestartRequiredMock() {
    return restartRequiredApi.getRestartRequired as unknown as Mock;
}

async function renderSettingsPage() {
    render(SettingsPage);
    await waitFor(() => expect(screen.getByLabelText("Update Interval")).not.toBeDisabled());
}

describe("settings page", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();

        contextMocks.settings = new SettingsProvider();
        contextMocks.toast.success.mockClear();
        contextMocks.toast.error.mockClear();

        getConfigMock().mockResolvedValue(baseConfig());
        patchConfigMock().mockResolvedValue("success");
        getRestartRequiredMock().mockResolvedValue({ restart_required: false });
    });

    it("discards unsaved dashboard setting changes", async () => {
        await renderSettingsPage();
        const interval = screen.getByLabelText("Update Interval") as HTMLInputElement;

        await fireEvent.input(interval, { target: { value: "5000" } });

        expect(screen.getByText("You have unsaved changes!")).toBeInTheDocument();

        await fireEvent.click(screen.getByRole("button", { name: /discard/i }));

        expect(interval.value).toBe("10000");
        expect(screen.queryByText("You have unsaved changes!")).not.toBeInTheDocument();
        expect(patchConfigMock()).not.toHaveBeenCalled();
    });

    it("saves changed proxy settings and reloads settings state", async () => {
        await renderSettingsPage();
        patchConfigMock().mockClear();
        getConfigMock().mockClear();

        await fireEvent.click(screen.getByRole("button", { name: /network/i }));
        await fireEvent.input(screen.getByLabelText("Proxy Listen"), {
            target: { value: ":7777" },
        });
        await fireEvent.click(screen.getByRole("button", { name: /save changes/i }));

        await waitFor(() =>
            expect(patchConfigMock()).toHaveBeenCalledWith("proxy.listen", ":7777"),
        );
        expect(getConfigMock()).toHaveBeenCalled();
        expect(contextMocks.toast.success).toHaveBeenCalledWith("Settings saved successfully.");
    });

    it("shows restart-required state after saving a restart-required setting", async () => {
        getRestartRequiredMock()
            .mockResolvedValueOnce({ restart_required: false })
            .mockResolvedValue({ restart_required: true });
        patchConfigMock().mockResolvedValue("restart required");

        await renderSettingsPage();

        await fireEvent.click(screen.getByRole("button", { name: /cache/i }));
        await fireEvent.change(screen.getByLabelText("Cache Backend"), {
            target: { value: "file" },
        });
        await fireEvent.click(screen.getByRole("button", { name: /save changes/i }));

        await waitFor(() => expect(screen.getByText("Restart Required")).toBeInTheDocument());
        expect(contextMocks.settings?.proxySettings.needsRestart).toBe(true);
    });

    it("shows only cache settings relevant to the staged backend", async () => {
        await renderSettingsPage();
        await fireEvent.click(screen.getByRole("button", { name: /cache/i }));

        expect(screen.getByLabelText("Cache Directory")).toBeInTheDocument();
        expect(screen.getByLabelText("Memory Budget (%)")).toBeInTheDocument();
        expect(screen.getByLabelText("Demote Idle Memory After")).toBeInTheDocument();

        await fireEvent.change(screen.getByLabelText("Cache Backend"), {
            target: { value: "memory" },
        });

        await waitFor(() =>
            expect(screen.queryByLabelText("Cache Directory")).not.toBeInTheDocument(),
        );
        expect(screen.getByLabelText("Memory Budget (%)")).toBeInTheDocument();
        expect(screen.queryByLabelText("Demote Idle Memory After")).not.toBeInTheDocument();

        await fireEvent.change(screen.getByLabelText("Cache Backend"), {
            target: { value: "file" },
        });

        await waitFor(() => expect(screen.getByLabelText("Cache Directory")).toBeInTheDocument());
        expect(screen.queryByLabelText("Memory Budget (%)")).not.toBeInTheDocument();
        expect(screen.queryByLabelText("Demote Idle Memory After")).not.toBeInTheDocument();
    });

    it("does not save hidden backend-specific cache settings", async () => {
        await renderSettingsPage();
        await fireEvent.click(screen.getByRole("button", { name: /cache/i }));

        await fireEvent.input(screen.getByLabelText("Demote Idle Memory After"), {
            target: { value: "1m" },
        });
        await fireEvent.change(screen.getByLabelText("Cache Backend"), {
            target: { value: "file" },
        });
        await waitFor(() =>
            expect(screen.queryByLabelText("Demote Idle Memory After")).not.toBeInTheDocument(),
        );
        await fireEvent.click(screen.getByRole("button", { name: /save changes/i }));

        await waitFor(() => expect(patchConfigMock()).toHaveBeenCalledWith("cache.type", "file"));
        expect(patchConfigMock()).not.toHaveBeenCalledWith("cache.hybrid.demote_after", "1m");
    });
});
