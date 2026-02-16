import { createContext } from "svelte";
import type { SettingsProvider } from "./providers/settings/settings-provider.svelte";
import type { MetricsProvider } from "./providers/metrics/metrics-provider.svelte";
import type { ToastProvider } from "./providers/toast/toast-provider.svelte";
import type { AuthProvider } from "./providers/auth/auth-provider.svelte";

export const [getSettingsProvider, setSettingsProvider] = createContext<SettingsProvider>();
export const [getMetricsProvider, setMetricsProvider] = createContext<MetricsProvider>();
export const [getToastProvider, setToastProvider] = createContext<ToastProvider>();
export const [getAuthProvider, setAuthProvider] = createContext<AuthProvider>();
