import { beforeEach, describe, expect, it, vi } from "vitest";
import { defaultDashboardLayout } from "$lib/dashboard/dashboard-layout";
import { DashboardSettings } from "./dashboard-settings.svelte";

vi.mock("$app/environment", () => ({
    browser: true,
}));

vi.mock("$lib/utils/logger", () => ({
    log: {
        debug: vi.fn(),
        error: vi.fn(),
    },
}));

describe("DashboardSettings", () => {
    beforeEach(() => {
        localStorage.clear();
        vi.clearAllMocks();
    });

    it("loads saved dashboard settings from localStorage", () => {
        localStorage.setItem(
            "dashboardConfig",
            JSON.stringify({
                updateInterval: 5000,
                layout: [{ id: "system-info", span: { width: 2, height: 2 } }],
            }),
        );

        const settings = new DashboardSettings();

        expect(settings.fields.updateInterval).toBe(5000);
        expect(settings.fields.layout[0]).toEqual({
            id: "system-info",
            span: { width: 2, height: 2 },
            mobileSpan: { width: 1 },
        });
    });

    it("saves dashboard settings to localStorage", () => {
        const settings = new DashboardSettings();

        settings.fields.updateInterval = 2500;
        settings.save();

        const saved = JSON.parse(localStorage.getItem("dashboardConfig") ?? "{}");
        expect(saved.updateInterval).toBe(2500);
        expect(saved.layout).toEqual(defaultDashboardLayout());
    });

    it("reloads dashboard settings from localStorage", async () => {
        const settings = new DashboardSettings();

        localStorage.setItem(
            "dashboardConfig",
            JSON.stringify({
                updateInterval: 7500,
                layout: [{ id: "cache-storage", span: { width: 4, height: 1 } }],
            }),
        );
        await settings.reload();

        expect(settings.fields.updateInterval).toBe(7500);
        expect(settings.fields.layout[0]).toEqual({
            id: "cache-storage",
            span: { width: 4, height: 1 },
            mobileSpan: { width: 2, height: 2 },
        });
    });

    it("keeps defaults when stored dashboard settings are invalid JSON", () => {
        localStorage.setItem("dashboardConfig", "{");

        const settings = new DashboardSettings();

        expect(settings.fields.updateInterval).toBe(10000);
        expect(settings.fields.layout).toEqual(defaultDashboardLayout());
    });
});
