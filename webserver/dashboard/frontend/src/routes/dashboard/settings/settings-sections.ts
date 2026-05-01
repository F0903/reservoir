import {
    patchConfig,
    type ConfigPropPath,
    type ConfigPropValue,
} from "$lib/api/objects/config/config";
import Dropdown from "$lib/components/ui/input/Dropdown.svelte";
import NumberInput from "$lib/components/ui/input/NumberInput.svelte";
import PercentInput from "$lib/components/ui/input/PercentInput.svelte";
import TextInput from "$lib/components/ui/input/TextInput.svelte";
import Toggle from "$lib/components/ui/input/Toggle.svelte";
import type { SettingsProvider } from "$lib/providers/settings/settings-provider.svelte";
import { parseByteString } from "$lib/utils/bytestring";
import { log } from "$lib/utils/logger";
import { Database, FileText, Globe, PanelsTopLeft } from "@lucide/svelte";
import type { Component, ComponentProps } from "svelte";

const optionalStringPattern = "^.*$";
const stringPattern = "^.+$";
const bytesizePattern = "^(\\d+)([BKMGT])$"; // eg. 100B, 1K, 1M, 1G, 1T
const durationPattern = "^(?:\\+|-)?(?:(?:\\d+(?:\\.\\d+)?|\\.\\d+)(?:ns|us|\\u00B5s|ms|s|m|h))+$"; // eg. 100ms, 1s, 1m, 1h
const ipPortPattern =
    "^((?:(?:\\d{1,3}\\.){3}\\d{1,3}|\\[[0-9A-Fa-f:.]+(?:%[A-Za-z0-9._\\-]+)?\\])|(localhost))?:\\d{1,5}$"; // IP:port or [IPv6]:port

export const tabs = [
    { id: "dashboard", label: "Dashboard", icon: PanelsTopLeft },
    { id: "network", label: "Network", icon: Globe },
    { id: "cache", label: "Cache", icon: Database },
    { id: "logging", label: "Logging", icon: FileText },
] as const;

export type TabId = (typeof tabs)[number]["id"];

export type SettingInputInstance = {
    commit: () => Promise<void>;
    reset: () => Promise<void>;
    hasDiverged: () => boolean;
};

type InputSection<
    C extends Component<CP, CE, "value">,
    CP extends Record<string, unknown> = { [key: string]: unknown },
    CE extends Record<string, unknown> = { [key: string]: unknown },
    O = ComponentProps<C>["value"],
> = {
    InputComponent: C;
    get: () => ComponentProps<C>["value"];
    valueTransform?: (_val: ComponentProps<C>["value"]) => O;
    commit: (_val: O) => Promise<unknown>;
    label: string;
    pattern?: string;
    tooltip?: string;
} & Omit<ComponentProps<C>, "value" | "label">;

type RegularInputProps<V> = {
    label: string;
    value: V;
};

type SetErrorProp = {
    setError: (_error: string) => void;
};

export type SettingsInput =
    | InputSection<typeof NumberInput, RegularInputProps<number>>
    | InputSection<typeof PercentInput, RegularInputProps<number>>
    | InputSection<typeof TextInput, RegularInputProps<string>, SetErrorProp>
    | InputSection<typeof TextInput, RegularInputProps<string>, SetErrorProp, number>
    | InputSection<typeof Toggle, RegularInputProps<boolean>>
    | InputSection<typeof Dropdown, RegularInputProps<string> & { options: string[] }>;

export type SettingsSections = { [_K in TabId]: SettingsInput[][] };

async function sendConfigPatch<P extends ConfigPropPath>(
    settings: SettingsProvider,
    propName: P,
    value: ConfigPropValue<P>,
) {
    const status = await patchConfig(propName, value);
    log.debug(`Patched config ${propName} with value ${value}, status: ${status}`);
    if (status === "restart required") {
        settings.proxySettings.needsRestart = true;
    }
}

function createConfigCommit(settings: SettingsProvider) {
    return <P extends ConfigPropPath>(propName: P) =>
        async (value: ConfigPropValue<P>) =>
            sendConfigPatch(settings, propName, value);
}

export function createInputInstances(
    sections: SettingsSections,
): Record<TabId, (SettingInputInstance | undefined)[][]> {
    const createTabInstances = (tabSections: SettingsInput[][]) =>
        tabSections.map((section) => new Array<SettingInputInstance | undefined>(section.length));

    return {
        dashboard: createTabInstances(sections.dashboard),
        network: createTabInstances(sections.network),
        cache: createTabInstances(sections.cache),
        logging: createTabInstances(sections.logging),
    };
}

export function createSettingsSections(settings: SettingsProvider): SettingsSections {
    const commit = createConfigCommit(settings);
    const proxyConfig = () => settings.proxySettings.fields.proxy;
    const webserverConfig = () => settings.proxySettings.fields.webserver;
    const cacheConfig = () => settings.proxySettings.fields.cache;
    const loggingConfig = () => settings.proxySettings.fields.logging;

    return {
        dashboard: [
            [
                {
                    InputComponent: NumberInput,
                    get: () => settings.dashboardSettings.fields.updateInterval,
                    commit: async (val: number) => {
                        settings.dashboardSettings.fields.updateInterval = val;
                        settings.dashboardSettings.save();
                    },
                    label: "Update Interval",
                    min: 500,
                    tooltip: "Interval in milliseconds for dashboard data updates.",
                },
            ],
        ],
        network: [
            [
                {
                    InputComponent: TextInput,
                    get: () => proxyConfig().listen,
                    commit: commit("proxy.listen"),
                    label: "Proxy Listen",
                    pattern: ipPortPattern,
                    tooltip: "IP and port for the proxy server.",
                },
                {
                    InputComponent: TextInput,
                    get: () => webserverConfig().listen,
                    commit: commit("webserver.listen"),
                    label: "Webserver Listen",
                    pattern: ipPortPattern,
                    tooltip: "IP and port for the internal web server.",
                },
            ],
            [
                {
                    InputComponent: TextInput,
                    get: () => proxyConfig().ca_cert,
                    commit: commit("proxy.ca_cert"),
                    label: "CA Certificate Path",
                    pattern: stringPattern,
                },
                {
                    InputComponent: TextInput,
                    get: () => proxyConfig().ca_key,
                    commit: commit("proxy.ca_key"),
                    label: "CA Key Path",
                    pattern: stringPattern,
                },
            ],
            [
                {
                    InputComponent: Toggle,
                    get: () => proxyConfig().upstream_default_https,
                    commit: commit("proxy.upstream_default_https"),
                    label: "Upstream Default HTTPS",
                },
                {
                    InputComponent: Toggle,
                    get: () => proxyConfig().retry_on_range_416,
                    commit: commit("proxy.retry_on_range_416"),
                    label: "Retry on Range 416",
                },
            ],
        ],
        cache: [
            [
                {
                    InputComponent: TextInput,
                    get: () => proxyConfig().cache_policy.default_max_age,
                    commit: commit("proxy.cache_policy.default_max_age"),
                    label: "Default Max Age",
                    pattern: durationPattern,
                    tooltip:
                        "Freshness lifetime Reservoir applies to cached package responses when using the default policy.",
                },
                {
                    InputComponent: Toggle,
                    get: () => proxyConfig().cache_policy.force_default_max_age,
                    commit: commit("proxy.cache_policy.force_default_max_age"),
                    label: "Force Default Max Age",
                    tooltip:
                        "Always use Reservoir's default max age instead of upstream freshness metadata.",
                },
                {
                    InputComponent: Toggle,
                    get: () => proxyConfig().cache_policy.ignore_cache_control,
                    commit: commit("proxy.cache_policy.ignore_cache_control"),
                    label: "Ignore Cache Control",
                    tooltip:
                        "Allow package responses to be cached even when upstream cache-control would normally prevent it.",
                },
            ],
            [
                {
                    InputComponent: Dropdown,
                    get: () => cacheConfig().type,
                    commit: commit("cache.type"),
                    label: "Storage Type",
                    options: ["memory", "file"],
                    tooltip: "Cache backend to use. Changing this requires a restart.",
                },
                {
                    InputComponent: TextInput,
                    get: () => cacheConfig().file.dir,
                    commit: commit("cache.file.dir"),
                    label: "Cache Directory",
                    pattern: stringPattern,
                    tooltip: "Directory for file-cache storage. Changing this requires a restart.",
                },
            ],
            [
                {
                    InputComponent: TextInput,
                    get: () => String(cacheConfig().max_cache_size),
                    valueTransform: (val: string) => parseByteString(val),
                    commit: commit("cache.max_cache_size"),
                    label: "Max Cache Size",
                    pattern: bytesizePattern,
                    tooltip:
                        "Maximum bytes Reservoir may keep in cache. This applies without a restart.",
                },
                {
                    InputComponent: TextInput,
                    get: () => cacheConfig().cleanup_interval,
                    commit: commit("cache.cleanup_interval"),
                    label: "Cleanup Interval",
                    pattern: durationPattern,
                    tooltip:
                        "How often Reservoir removes expired entries and trims oversized cache data.",
                },
                {
                    InputComponent: PercentInput,
                    get: () => cacheConfig().memory.memory_budget_percent,
                    commit: commit("cache.memory.memory_budget_percent"),
                    label: "Memory Budget (%)",
                    tooltip: "Maximum percentage of system memory the memory cache may use.",
                },
            ],
        ],
        logging: [
            [
                {
                    InputComponent: Dropdown,
                    options: ["DEBUG", "INFO", "WARN", "ERROR"],
                    get: () => loggingConfig().level,
                    commit: commit("logging.level"),
                    label: "Log Level",
                },
                {
                    InputComponent: Toggle,
                    get: () => loggingConfig().to_stdout,
                    commit: commit("logging.to_stdout"),
                    label: "Log to Stdout",
                },
            ],
            [
                {
                    InputComponent: TextInput,
                    get: () => loggingConfig().file,
                    commit: commit("logging.file"),
                    label: "Log File Path",
                    pattern: optionalStringPattern,
                },
                {
                    InputComponent: TextInput,
                    get: () => String(loggingConfig().max_size),
                    valueTransform: (val: string) => parseByteString(val),
                    commit: commit("logging.max_size"),
                    label: "Max File Size",
                    pattern: bytesizePattern,
                },
            ],
        ],
    };
}
