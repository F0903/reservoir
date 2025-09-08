<script lang="ts">
    import { patchConfig } from "$lib/api/objects/config/config.svelte";
    import PageTitle from "$lib/components/ui/PageTitle.svelte";
    import SettingInput from "$lib/components/ui/settings/SettingInput.svelte";
    import TextInput from "$lib/components/ui/TextInput.svelte";
    import VerticalSpacer from "$lib/components/ui/VerticalSpacer.svelte";
    import type { SettingsProvider } from "$lib/providers/settings/settings-provider.svelte";
    import type { ToastProvider } from "$lib/providers/toast.svelte";
    import { parseByteString } from "$lib/utils/format";
    import { log } from "$lib/utils/logger";
    import { getContext, onMount } from "svelte";

    const settings = getContext("settings") as SettingsProvider;
    const toast = getContext("toast") as ToastProvider;

    const inputSections = [
        [
            // Dashboard section
            {
                InputComponent: TextInput,
                getSetting: () => settings.dashboardConfig.fields.updateInterval,
                setSetting: (val: number) => (settings.dashboardConfig.fields.updateInterval = val),
                settingTransform: (val: string) => parseInt(val),
                label: "Dashboard Update Interval",
                pattern: "^\\d+$", // Only digits
                min: 500,
                tooltip:
                    "The interval at which the dashboard updates its data from the API in milliseconds.",
            },
        ],
        // Logging section
        [
            {
                InputComponent: TextInput,
                getSetting: () => settings.proxySettings.fields.logLevel,
                setSetting: async (val: string) => await patchConfig("log_level", val),
                settingTransform: (val: string) => val,
                label: "Log Level",
                pattern: "^(DEBUG|INFO|WARN|ERROR)$", // One of these values
                tooltip:
                    "The minimum level of logs to be recorded. Options are: DEBUG, INFO, WARN, ERROR.",
            },
            {
                InputComponent: TextInput,
                getSetting: () => settings.proxySettings.fields.logFile,
                setSetting: async (val: string) => await patchConfig("log_file", val),
                settingTransform: (val: string) => val,
                label: "Log File Path",
                pattern: ".?", // Any string
                tooltip:
                    "The file path where the application log will be stored. Leave empty to disable file logging.",
            },
            {
                InputComponent: TextInput,
                getSetting: () => settings.proxySettings.fields.logFileMaxSize,
                setSetting: async (val: number) => await patchConfig("log_file_max_size", val),
                settingTransform: (val: string) => parseByteString(val),
                label: "Log File Max Size",
                pattern: "^(\\d+)([BKMGT])$", // Only digits
                tooltip: "The maximum size (in bytes) of the log file before it is rotated.",
            },
            {
                InputComponent: TextInput,
                getSetting: () => settings.proxySettings.fields.logFileMaxBackups,
                setSetting: async (val: number) => await patchConfig("log_file_max_backups", val),
                settingTransform: (val: string) => parseInt(val),
                label: "Log File Max Backups",
                pattern: "^\\d+$", // Only digits
                tooltip:
                    "The maximum number of rotated log files to keep. Older files will be deleted.",
            },
            {
                InputComponent: TextInput,
                getSetting: () => settings.proxySettings.fields.logFileCompress,
                setSetting: async (val: boolean) => await patchConfig("log_file_compress", val),
                settingTransform: (val: string) => val === "true",
                label: "Log File Compression",
                pattern: "^(true|false)$",
                tooltip: "Whether to compress log files when they are rotated.",
            },
            {
                InputComponent: TextInput,
                getSetting: () => settings.proxySettings.fields.logToStdout,
                setSetting: async (val: boolean) => await patchConfig("log_to_stdout", val),
                settingTransform: (val: string) => val === "true",
                label: "Log to Stdout",
                pattern: "^(true|false)$",
                tooltip: "Whether to also log to standard output (console).",
            },
        ],
    ];

    const inputComponents: SettingInput<any>[] = $state([]);
    let hasChanges = $state(false);

    onMount(() => {
        settings.proxySettings.reload();
        settings.dashboardConfig.reload();
    });

    $effect(() => {
        if (!hasChanges) {
            toast.close();
            log.debug("No unsaved changes, closing toast if open.");
            return;
        }

        log.debug("Unsaved changes detected, prompting user to save or discard.");
        toast.show({
            type: "action",
            message: "You have unsaved changes!",
            negativeText: "Discard",
            positiveText: "Save",
            onNegative: discardChanges,
            onPositive: applyChanges,
        });
    });

    function onChange(different: boolean) {
        hasChanges = different;
    }

    async function saveInputs() {
        await Promise.all(inputComponents.map((input) => input.save()));
    }

    async function resetInputs() {
        await Promise.all(inputComponents.map((input) => input.reset()));
    }

    async function applyChanges() {
        await saveInputs();
        toast.close();
        await settings.proxySettings.reload();
        await resetInputs();

        log.debug("Settings have been saved.");
    }

    async function discardChanges() {
        await resetInputs();
        toast.close();
        log.debug("Changes have been discarded.");
    }
</script>

<PageTitle>Settings</PageTitle>
<div class="inputs">
    {#each inputSections as section, i}
        {#each section as input, j}
            <SettingInput bind:this={inputComponents[i + j]} {...input} {onChange} />
        {/each}
        {#if i < inputSections.length - 1}
            <VerticalSpacer --spacer-color="var(--secondary-700)" />
        {/if}
    {/each}
</div>

<style>
    .inputs {
        width: fit-content;
    }
</style>
