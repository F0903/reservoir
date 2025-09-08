import { DashboardSettings } from "./dashboard-settings.svelte";
import { ProxySettings } from "./proxy-settings.svelte";

// An interface that all settings objects must implement
export interface Settings {
    reload(): Promise<void>;
}

// A container for all settings objects. Provided as context to all components.
export class SettingsProvider {
    dashboardConfig: DashboardSettings = new DashboardSettings();
    proxySettings: ProxySettings = new ProxySettings();
}
