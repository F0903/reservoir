import { DashboardSettings } from "./dashboard-settings.svelte";
import { ProxySettings } from "./proxy-settings.svelte";

export interface Settings {
    reload(): Promise<void>;
}

// A container for all settings objects. Provided as context to all components.
export class SettingsProvider {
    dashboardSettings: DashboardSettings = new DashboardSettings();
    proxySettings: ProxySettings = new ProxySettings();
}
