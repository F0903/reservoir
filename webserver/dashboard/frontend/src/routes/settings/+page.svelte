<script lang="ts">
    import PageTitle from "$lib/components/ui/PageTitle.svelte";
    import SettingInput from "$lib/components/ui/settings/SettingInput.svelte";
    import TextInput from "$lib/components/ui/TextInput.svelte";
    import type { SettingsProvider } from "$lib/providers/settings.svelte";
    import type { ToastProvider } from "$lib/providers/toast.svelte";
    import { log } from "$lib/utils/logger";
    import { getContext } from "svelte";

    const settings = getContext("settings") as SettingsProvider;
    const toast = getContext("toast") as ToastProvider;

    const inputs = [
        {
            InputComponent: TextInput,
            settingName: "updateInterval",
            settingObject: settings.dashboardConfig,
            label: "Dashboard Update Interval",
            pattern: "\\d+", // Only digits
            onChange: onChange,
            min: 500,
            tooltip:
                "The interval at which the dashboard updates its data from the API in milliseconds.",
        },
    ];

    const inputComponents: SettingInput<any>[] = $state([]);
    let hasChanges = $state(false);

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

    function applyChanges() {
        inputComponents.forEach((input) => input.save());
        toast.close();
        settings.dashboardConfig.save();
    }

    function discardChanges() {
        inputComponents.forEach((input) => input.reset());
        toast.close();
    }
</script>

<PageTitle>Settings</PageTitle>
{#each inputs as input, i}
    <SettingInput bind:this={inputComponents[i]} {...input} />
{/each}
