import { beforeEach, describe, expect, it, vi } from "vitest";
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
        localStorage.setItem("dashboardConfig", JSON.stringify({ updateInterval: 5000 }));

        const settings = new DashboardSettings();

        expect(settings.fields.updateInterval).toBe(5000);
    });

    it("saves dashboard settings to localStorage", () => {
        const settings = new DashboardSettings();

        settings.fields.updateInterval = 2500;
        settings.save();

        expect(JSON.parse(localStorage.getItem("dashboardConfig") ?? "{}")).toEqual({
            updateInterval: 2500,
        });
    });

    it("reloads dashboard settings from localStorage", async () => {
        const settings = new DashboardSettings();

        localStorage.setItem("dashboardConfig", JSON.stringify({ updateInterval: 7500 }));
        await settings.reload();

        expect(settings.fields.updateInterval).toBe(7500);
    });

    it("keeps defaults when stored dashboard settings are invalid JSON", () => {
        localStorage.setItem("dashboardConfig", "{");

        const settings = new DashboardSettings();

        expect(settings.fields.updateInterval).toBe(10000);
    });
});
