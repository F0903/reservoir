<script lang="ts">
    import PageTitle from "$lib/components/ui/PageTitle.svelte";
    import SettingInput from "$lib/components/ui/settings/SettingInput.svelte";
    import TextInput from "$lib/components/ui/input/TextInput.svelte";
    import VerticalSpacer from "$lib/components/ui/VerticalSpacer.svelte";
    import type { SettingsProvider } from "$lib/providers/settings/settings-provider.svelte";
    import type { ToastHandle, ToastProvider } from "$lib/providers/toast-provider.svelte";
    import { log } from "$lib/utils/logger";
    import { getContext, onMount, type Component } from "svelte";
    import Toggle from "$lib/components/ui/input/Toggle.svelte";
    import Dropdown from "$lib/components/ui/input/Dropdown.svelte";
    import { parseByteString } from "$lib/utils/bytestring";
    import { patchConfig } from "$lib/api/objects/config/config";

    const settings = getContext("settings") as SettingsProvider;
    const toast = getContext("toast") as ToastProvider;

    const optionalStringPattern = "^.*$";
    const stringPattern = "^.+$";
    const boolPattern = "^(true|false)$";
    const intPattern = "^\\d+$";
    const bytesizePattern = "^(\\d+)([BKMGT])$";
    const durationPattern =
        "^(?:\\+|-)?(?:(?:\\d+(?:\\.\\d+)?|\\.\\d+)(?:ns|us|\\u00B5s|ms|s|m|h))+$";
    const ipPortPattern =
        "^((?:(?:\\d{1,3}\\.){3}\\d{1,3}|\\[[0-9A-Fa-f:.]+(?:%[A-Za-z0-9._\\-]+)?\\])|(localhost))?:\\d{1,5}$"; // IP:port or [IPv6]:port
    const logLevelPattern = "^(DEBUG|INFO|WARN|ERROR)$"; // One of these values

    type InputSection = {
        InputComponent: Component<any, any, "value">;
        get: () => any;
        valueTransform?: (val: any) => any;
        commit: (val: any) => Promise<any>;
        label: string;
        pattern: string;
        tooltip?: string;
        [key: string]: any; // Allow additional props for the input component
    };

    // Thin wrapper so we can show a toast if a restart is required
    async function sendPatch(propName: string, value: unknown) {
        const status = await patchConfig(propName, value);
        log.debug(`Patched config ${propName} with value ${value}, status: ${status}`);
        if (status === "restart required") {
            settings.proxySettings.needsRestart = true;
            toast.show({
                type: "info",
                message: "Restart required to apply changes.",
                durationMs: 10000,
            });
        }
    }

    const inputSections: InputSection[][] = [
        // Dashboard section
        [
            {
                InputComponent: TextInput,
                get: () => settings.dashboardSettings.fields.updateInterval,
                commit: async (val: number) =>
                    (settings.dashboardSettings.fields.updateInterval = val),
                valueTransform: (val: string) => parseInt(val),
                label: "Dashboard Update Interval",
                pattern: intPattern,
                min: 500,
                tooltip:
                    "The interval at which the dashboard updates its data from the API in milliseconds.",
            },
        ],
        // Main proxy section
        [
            {
                InputComponent: TextInput,
                get: () => settings.proxySettings.fields.proxy_listen,
                commit: async (val: string) => await sendPatch("proxy_listen", val),
                label: "Proxy Listen",
                pattern: ipPortPattern,
                tooltip: "The IP address and port that the proxy server will bind to.",
            },
            {
                InputComponent: TextInput,
                get: () => settings.proxySettings.fields.ca_cert,
                commit: async (val: string) => await sendPatch("ca_cert", val),
                label: "CA Certificate",
                pattern: stringPattern,
                tooltip: "The path to the CA certificate for the proxy server.",
            },
            {
                InputComponent: TextInput,
                get: () => settings.proxySettings.fields.ca_key,
                commit: async (val: string) => await sendPatch("ca_key", val),
                label: "CA Key",
                pattern: stringPattern,
                tooltip: "The path to the CA private key for the proxy server.",
            },
            {
                InputComponent: Toggle,
                get: () => settings.proxySettings.fields.upstream_default_https,
                commit: async (val: boolean) => await sendPatch("upstream_default_https", val),
                label: "Upstream Default HTTPS",
                pattern: boolPattern,
                tooltip:
                    "If true, the proxy will always send HTTPS instead of HTTP to the upstream server.",
            },
        ],
        // Webserver section
        [
            {
                InputComponent: TextInput,
                get: () => settings.proxySettings.fields.webserver_listen,
                commit: async (val: string) => await sendPatch("webserver_listen", val),
                label: "Webserver Listen",
                pattern: ipPortPattern,
                tooltip: "The IP address and port that the web server will bind to.",
            },
            {
                InputComponent: Toggle,
                get: () => settings.proxySettings.fields.dashboard_disabled,
                commit: async (val: boolean) => await sendPatch("dashboard_disabled", val),
                label: "Dashboard Disabled",
                pattern: boolPattern,
                tooltip: "Whether the dashboard is disabled.",
            },
            {
                InputComponent: Toggle,
                get: () => settings.proxySettings.fields.api_disabled,
                commit: async (val: boolean) => await sendPatch("api_disabled", val),
                label: "API Disabled",
                pattern: boolPattern,
                tooltip:
                    "Whether the API is disabled. The API is required for the dashboard to function.",
            },
        ],
        // Cache section
        [
            {
                InputComponent: TextInput,
                get: () => settings.proxySettings.fields.cache_dir,
                commit: async (val: string) => await sendPatch("cache_dir", val),
                label: "Cache Directory",
                pattern: stringPattern,
                tooltip: "The directory where cached files are stored.",
            },
            {
                InputComponent: Toggle,
                get: () => settings.proxySettings.fields.ignore_cache_control,
                commit: async (val: boolean) => await sendPatch("ignore_cache_control", val),
                label: "Ignore Cache Control",
                pattern: boolPattern,
                tooltip: "Whether to ignore Cache-Control headers from the client.",
            },
            {
                InputComponent: TextInput,
                get: () => settings.proxySettings.fields.max_cache_size,
                valueTransform: (val: string) => parseByteString(val),
                commit: async (val: number) => await sendPatch("max_cache_size", val),
                label: "Max Cache Size",
                pattern: bytesizePattern,
                tooltip: "The maximum size of the cache. You can use suffixes like B, K, M, G, T.",
            },
            {
                InputComponent: TextInput,
                get: () => settings.proxySettings.fields.default_cache_max_age,
                commit: async (val: string) => await sendPatch("default_cache_max_age", val),
                label: "Default Cache Max Age",
                pattern: durationPattern,
                tooltip:
                    "The default cache max age to use if the upstream response does not specify a Cache-Control or Expires header.",
            },
            {
                InputComponent: Toggle,
                get: () => settings.proxySettings.fields.force_default_cache_max_age,
                commit: async (val: boolean) => await sendPatch("force_default_cache_max_age", val),
                label: "Force Default Cache Max Age",
                pattern: boolPattern,
                tooltip:
                    "If true, always use the default cache max age even if the upstream response has a Cache-Control or Expires header.",
            },
            {
                InputComponent: TextInput,
                get: () => settings.proxySettings.fields.cache_cleanup_interval,
                commit: async (val: string) => await sendPatch("cache_cleanup_interval", val),
                label: "Cache Cleanup Interval",
                pattern: durationPattern,
                tooltip:
                    "The interval at which the cache will be cleaned up to remove expired entries.",
            },
        ],
        // Logging section
        [
            {
                InputComponent: Dropdown,
                options: ["DEBUG", "INFO", "WARN", "ERROR"],
                get: () => settings.proxySettings.fields.log_level,
                commit: async (val: string) => await sendPatch("log_level", val),
                label: "Log Level",
                pattern: logLevelPattern,
                tooltip:
                    "The minimum level of logs to be recorded. Options are: DEBUG, INFO, WARN, ERROR.",
            },
            {
                InputComponent: TextInput,
                get: () => settings.proxySettings.fields.log_file,
                commit: async (val: string) => await sendPatch("log_file", val),
                label: "Log File Path",
                pattern: optionalStringPattern,
                tooltip:
                    "The file path where the application log will be stored. Leave empty to disable file logging.",
            },
            {
                InputComponent: TextInput,
                get: () => settings.proxySettings.fields.log_file_max_size,
                valueTransform: (val: string) => parseByteString(val),
                commit: async (val: number) => await sendPatch("log_file_max_size", val),
                label: "Log File Max Size",
                pattern: bytesizePattern,
                tooltip:
                    "The maximum size (in bytes) of the log file before it is rotated. You can use suffixes like B, K, M, G, T.",
            },
            {
                InputComponent: TextInput,
                get: () => settings.proxySettings.fields.log_file_max_backups,
                valueTransform: (val: string) => parseInt(val),
                commit: async (val: number) => await sendPatch("log_file_max_backups", val),
                label: "Log File Max Backups",
                pattern: intPattern,
                tooltip:
                    "The maximum number of rotated log files to keep. Older files will be deleted.",
            },
            {
                InputComponent: Toggle,
                get: () => settings.proxySettings.fields.log_file_compress,
                commit: async (val: boolean) => await sendPatch("log_file_compress", val),
                label: "Log File Compression",
                pattern: boolPattern,
                tooltip: "Whether to compress log files when they are rotated.",
            },
            {
                InputComponent: Toggle,
                get: () => settings.proxySettings.fields.log_to_stdout,
                commit: async (val: boolean) => await sendPatch("log_to_stdout", val),
                label: "Log to Stdout",
                pattern: boolPattern,
                tooltip: "Whether to also log to standard output (console).",
            },
        ],
    ];

    const inputComponents: SettingInput<any, any>[][] = $state(
        // Initialize a 2D array (an array for each section) to hold references to SettingInput components
        inputSections.map(() => []),
    );
    let hasChanges = $state(false);

    let inputsDisabled = $state(true); // Disable inputs until settings are loaded

    let changesToast: ToastHandle | null = null;

    onMount(async () => {
        log.debug("Loading settings for settings page...");
        await Promise.all([settings.proxySettings.reload(), settings.dashboardSettings.reload()]);
        await resetInputs();
        inputsDisabled = false;
        log.debug("Settings loaded, inputs enabled.");
    });

    $effect(() => {
        if (!hasChanges) {
            changesToast?.close();
            log.debug("No unsaved changes, closing toast if open.");
            return;
        }

        log.debug("Unsaved changes detected, prompting user to save or discard.");
        changesToast = toast.show({
            type: "action",
            message: "You have unsaved changes!",
            negativeText: "Discard",
            positiveText: "Save",
            onNegative: discardChanges,
            onPositive: commitChanges,
        });
    });

    function onChange(different: boolean) {
        hasChanges = different;
    }

    async function commitInputs() {
        // Consider sending a batch update instead at some point.
        await Promise.all(
            inputComponents
                .flat()
                .filter((input) => input)
                .map((input) => input.commit()),
        );
    }

    async function resetInputs() {
        await Promise.all(
            inputComponents
                .flat()
                .filter((input) => input)
                .map((input) => input.reset()),
        );
    }

    async function commitChanges() {
        try {
            await commitInputs();
            log.debug("Settings have been committed.");
        } finally {
            changesToast?.close();
        }
        await settings.proxySettings.reload();
        await resetInputs();
    }

    async function discardChanges() {
        try {
            await resetInputs();
            log.debug("Changes have been discarded.");
        } finally {
            changesToast?.close();
        }
    }
</script>

<PageTitle>Settings</PageTitle>
{#if settings.proxySettings.needsRestart}
    <span class="restart-warning"
        >Changes have been made that require a restart to take effect.</span
    >
{/if}
<div class="inputs">
    {#each inputSections as section, i}
        {#each section as input, j}
            <SettingInput
                bind:this={inputComponents[i][j]}
                {...input}
                {onChange}
                disabled={inputsDisabled}
            />
        {/each}
        {#if i < inputSections.length - 1}
            <VerticalSpacer --spacer-color="var(--secondary-700)" />
        {/if}
    {/each}
</div>

<style>
    .inputs {
        display: flex;
        flex-direction: column;
        gap: 0px;
        align-items: flex-start;

        width: fit-content;
    }
</style>
