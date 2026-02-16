<script lang="ts">
    import PageTitle from "$lib/components/ui/PageTitle.svelte";
    import SettingInput from "$lib/components/ui/settings/SettingInput.svelte";
    import TextInput from "$lib/components/ui/input/TextInput.svelte";
    import VerticalSpacer from "$lib/components/ui/VerticalSpacer.svelte";
    import { log } from "$lib/utils/logger";
    import { onMount, type Component } from "svelte";
    import Toggle from "$lib/components/ui/input/Toggle.svelte";
    import Dropdown from "$lib/components/ui/input/Dropdown.svelte";
    import { parseByteString } from "$lib/utils/bytestring";
    import { patchConfig } from "$lib/api/objects/config/config";
    import NumberInput from "$lib/components/ui/input/NumberInput.svelte";
    import PercentInput from "$lib/components/ui/input/PercentInput.svelte";
    import { getSettingsProvider, getToastProvider } from "$lib/context";
    import {
        Globe,
        Database,
        FileText,
        PanelsTopLeft,
        Save,
        RotateCcw,
        RefreshCw,
        TriangleAlert,
    } from "@lucide/svelte";
    import Button from "$lib/components/ui/input/Button.svelte";
    import { fade, slide } from "svelte/transition";
    import type { ComponentProps } from "svelte";
    import Tabs from "$lib/components/ui/Tabs.svelte";

    const settings = getSettingsProvider();
    const toast = getToastProvider();

    const optionalStringPattern = "^.*$";
    const stringPattern = "^.+$";
    const bytesizePattern = "^(\\d+)([BKMGT])$"; // eg. 100B, 1K, 1M, 1G, 1T
    const durationPattern =
        "^(?:\\+|-)?(?:(?:\\d+(?:\\.\\d+)?|\\.\\d+)(?:ns|us|\\u00B5s|ms|s|m|h))+$"; // eg. 100ms, 1s, 1m, 1h
    const ipPortPattern =
        "^((?:(?:\\d{1,3}\\.){3}\\d{1,3}|\\[[0-9A-Fa-f:.]+(?:%[A-Za-z0-9._\\-]+)?\\])|(localhost))?:\\d{1,5}$"; // IP:port or [IPv6]:port

    const tabs = [
        { id: "dashboard", label: "Dashboard", icon: PanelsTopLeft },
        { id: "network", label: "Network", icon: Globe },
        { id: "cache", label: "Cache", icon: Database },
        { id: "logging", label: "Logging", icon: FileText },
    ] as const;

    type TabId = (typeof tabs)[number]["id"];
    let activeTab = $state<TabId>("dashboard");

    async function sendPatch(propName: string, value: unknown) {
        const status = await patchConfig(propName, value);
        log.debug(`Patched config ${propName} with value ${value}, status: ${status}`);
        if (status === "restart required") {
            settings.proxySettings.needsRestart = true;
        }
    }

    type SettingInputInstance = {
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

    const sections: {
        [_K in (typeof tabs)[number]["id"]]: (
            | InputSection<typeof NumberInput, RegularInputProps<number>>
            | InputSection<typeof PercentInput, RegularInputProps<number>>
            | InputSection<typeof TextInput, RegularInputProps<string>, SetErrorProp>
            | InputSection<typeof TextInput, RegularInputProps<string>, SetErrorProp, number>
            | InputSection<typeof Toggle, RegularInputProps<boolean>>
            | InputSection<typeof Dropdown, RegularInputProps<string> & { options: string[] }>
        )[][];
    } = {
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
                    get: () => settings.proxySettings.fields.proxy_listen,
                    commit: async (val: string) => await sendPatch("proxy_listen", val),
                    label: "Proxy Listen",
                    pattern: ipPortPattern,
                    tooltip: "IP and port for the proxy server.",
                },
                {
                    InputComponent: TextInput,
                    get: () => settings.proxySettings.fields.webserver_listen,
                    commit: async (val: string) => await sendPatch("webserver_listen", val),
                    label: "Webserver Listen",
                    pattern: ipPortPattern,
                    tooltip: "IP and port for the internal web server.",
                },
            ],
            [
                {
                    InputComponent: TextInput,
                    get: () => settings.proxySettings.fields.ca_cert,
                    commit: async (val: string) => await sendPatch("ca_cert", val),
                    label: "CA Certificate Path",
                    pattern: stringPattern,
                },
                {
                    InputComponent: TextInput,
                    get: () => settings.proxySettings.fields.ca_key,
                    commit: async (val: string) => await sendPatch("ca_key", val),
                    label: "CA Key Path",
                    pattern: stringPattern,
                },
            ],
            [
                {
                    InputComponent: Toggle,
                    get: () => settings.proxySettings.fields.upstream_default_https,
                    commit: async (val: boolean) => await sendPatch("upstream_default_https", val),
                    label: "Upstream Default HTTPS",
                },
                {
                    InputComponent: Toggle,
                    get: () => settings.proxySettings.fields.retry_on_range_416,
                    commit: async (val: boolean) => await sendPatch("retry_on_range_416", val),
                    label: "Retry on Range 416",
                },
            ],
        ],
        cache: [
            [
                {
                    InputComponent: Dropdown,
                    get: () => settings.proxySettings.fields.cache_type,
                    commit: async (val: string) => await sendPatch("cache_type", val),
                    label: "Storage Type",
                    options: ["memory", "file"],
                },
                {
                    InputComponent: TextInput,
                    get: () => settings.proxySettings.fields.cache_dir,
                    commit: async (val: string) => await sendPatch("cache_dir", val),
                    label: "Cache Directory",
                    pattern: stringPattern,
                },
            ],
            [
                {
                    InputComponent: TextInput,
                    get: () => String(settings.proxySettings.fields.max_cache_size),
                    valueTransform: (val: string) => parseByteString(val),
                    commit: async (val: number) => await sendPatch("max_cache_size", val),
                    label: "Max Cache Size",
                    pattern: bytesizePattern,
                },
                {
                    InputComponent: PercentInput,
                    get: () => settings.proxySettings.fields.cache_memory_budget_percent,
                    commit: async (val: number) =>
                        await sendPatch("cache_memory_budget_percent", val),
                    label: "Memory Budget (%)",
                },
            ],
            [
                {
                    InputComponent: TextInput,
                    get: () => settings.proxySettings.fields.default_cache_max_age,
                    commit: async (val: string) => await sendPatch("default_cache_max_age", val),
                    label: "Default Max Age",
                    pattern: durationPattern,
                },
                {
                    InputComponent: Toggle,
                    get: () => settings.proxySettings.fields.ignore_cache_control,
                    commit: async (val: boolean) => await sendPatch("ignore_cache_control", val),
                    label: "Ignore Cache Control",
                },
            ],
        ],
        logging: [
            [
                {
                    InputComponent: Dropdown,
                    options: ["DEBUG", "INFO", "WARN", "ERROR"],
                    get: () => settings.proxySettings.fields.log_level,
                    commit: async (val: string) => await sendPatch("log_level", val),
                    label: "Log Level",
                },
                {
                    InputComponent: Toggle,
                    get: () => settings.proxySettings.fields.log_to_stdout,
                    commit: async (val: boolean) => await sendPatch("log_to_stdout", val),
                    label: "Log to Stdout",
                },
            ],
            [
                {
                    InputComponent: TextInput,
                    get: () => settings.proxySettings.fields.log_file,
                    commit: async (val: string) => await sendPatch("log_file", val),
                    label: "Log File Path",
                    pattern: optionalStringPattern,
                },
                {
                    InputComponent: TextInput,
                    get: () => String(settings.proxySettings.fields.log_file_max_size),
                    valueTransform: (val: string) => parseByteString(val),
                    commit: async (val: number) => await sendPatch("log_file_max_size", val),
                    label: "Max File Size",
                    pattern: bytesizePattern,
                },
            ],
        ],
    };

    const inputInstances: Record<string, (SettingInputInstance | undefined)[][]> = $state(
        Object.fromEntries(
            Object.entries(sections).map(([tabId, tabSections]) => [
                tabId,
                tabSections.map((section) => new Array(section.length).fill(undefined)),
            ]),
        ),
    );

    let hasChanges = $state(false);
    let saving = $state(false);
    let inputsDisabled = $state(true);

    onMount(async () => {
        await Promise.all([settings.proxySettings.reload(), settings.dashboardSettings.reload()]);
        await resetInputs();
        inputsDisabled = false;
    });

    function onChange(_different: boolean) {
        // Check if ANY input across all tabs has diverged
        const allInputs = Object.values(inputInstances).flat(2);
        hasChanges = allInputs.some((i) => i?.hasDiverged?.());
    }

    async function commitChanges() {
        saving = true;
        try {
            const allInputs = Object.values(inputInstances).flat(2);
            await Promise.all(allInputs.map((i) => i?.commit()));

            toast.success("Settings saved successfully.");
            await settings.proxySettings.reload();
            await resetInputs();
        } catch (e) {
            log.error("Failed to save settings:", e);
        } finally {
            saving = false;
        }
    }

    async function resetInputs() {
        const allInputs = Object.values(inputInstances).flat(2);
        await Promise.all(allInputs.map((i) => i?.reset()));
        hasChanges = false;
    }
</script>

<main class="settings-page">
    <div class="header">
        <PageTitle --pagetitle-margin-bottom="0">Settings</PageTitle>

        {#if settings.proxySettings.needsRestart}
            <div class="restart-badge" transition:fade>
                <TriangleAlert size={14} />
                <span>Restart Required</span>
            </div>
        {/if}
    </div>

    <div class="settings-container">
        <Tabs {tabs} bind:activeTab>
            {#snippet dashboard()}
                {@render pane("dashboard")}
            {/snippet}
            {#snippet network()}
                {@render pane("network")}
            {/snippet}
            {#snippet cache()}
                {@render pane("cache")}
            {/snippet}
            {#snippet logging()}
                {@render pane("logging")}
            {/snippet}
        </Tabs>
    </div>

    {#snippet pane(tabId: TabId)}
        {#each sections[tabId] as section, i (i)}
            <div class="settings-group">
                <div class="group-grid">
                    {#each section as input, j (input.label)}
                        <SettingInput
                            bind:this={inputInstances[tabId][i][j]}
                            {...input as Record<string, unknown>}
                            InputComponent={input.InputComponent as Component<
                                Record<string, unknown>,
                                Record<string, unknown>,
                                "value"
                            >}
                            get={input.get as () => unknown}
                            commit={input.commit as (_val: unknown) => Promise<unknown>}
                            disabled={inputsDisabled}
                            {onChange}
                        />
                    {/each}
                </div>
            </div>
            {#if i < sections[tabId].length - 1}
                <VerticalSpacer
                    --spacer-color="rgba(255,255,255,0.05)"
                    --spacer-margin="1.5rem -2.5rem"
                    --spacer-width="calc(100% + 5rem)"
                />
            {/if}
        {/each}
    {/snippet}

    {#if hasChanges}
        <div class="action-bar" transition:slide={{ axis: "y" }}>
            <div class="action-content">
                <div class="message">
                    <TriangleAlert size={20} />
                    <span>You have unsaved changes!</span>
                </div>
                <div class="buttons">
                    <Button
                        onClick={resetInputs}
                        disabled={saving}
                        --btn-background-color="transparent"
                        --btn-text-color="var(--text-400)"
                    >
                        <div class="btn-inner"><RotateCcw size={16} /> Discard</div>
                    </Button>
                    <Button onClick={commitChanges} disabled={saving}>
                        <div class="btn-inner">
                            {#if saving}
                                <RefreshCw size={16} class="spin" />
                                Saving...
                            {:else}
                                <Save size={16} />
                                Save Changes
                            {/if}
                        </div>
                    </Button>
                </div>
            </div>
        </div>
    {/if}
</main>

<style>
    .settings-page {
        height: 100%;
        display: flex;
        flex-direction: column;
        gap: 1rem;
        padding-bottom: 80px; /* Space for action bar */
    }

    .header {
        display: flex;
        align-items: center;
        gap: 1.5rem;
    }

    .restart-badge {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        background-color: var(--error-bg);
        color: var(--error-color);
        padding: 0.4rem 0.8rem;
        border-radius: 20px;
        font-size: 0.8rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        border: 1px solid var(--error-border);
    }

    .settings-container {
        display: flex;
        flex-direction: column;
        align-self: center;
        width: 100%;
        max-width: 1000px;
    }

    .group-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
        gap: 1.5rem 3rem;
        width: 100%;
    }

    .action-bar {
        position: fixed;
        bottom: 2rem;
        left: 50%;
        transform: translateX(-50%);
        width: calc(100% - 4rem);
        max-width: 800px;
        background-color: var(--primary-300);
        backdrop-filter: blur(10px);
        border: 1px solid var(--secondary-500);
        border-radius: 16px;
        padding: 1rem 1.5rem;
        box-shadow: 0 10px 40px rgba(0, 0, 0, 0.4);
        z-index: 100;
    }

    .action-content {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .message {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        color: var(--secondary-300);
        font-weight: 600;
    }

    .buttons {
        display: flex;
        gap: 1rem;
    }

    .btn-inner {
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }

    @keyframes spin {
        from {
            transform: rotate(0deg);
        }
        to {
            transform: rotate(360deg);
        }
    }

    :global(.spin) {
        animation: spin 1s linear infinite;
    }
</style>
