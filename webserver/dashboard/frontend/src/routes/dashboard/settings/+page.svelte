<script lang="ts">
    import PageTitle from "$lib/components/ui/PageTitle.svelte";
    import SettingInput from "$lib/components/ui/settings/SettingInput.svelte";
    import VerticalSpacer from "$lib/components/ui/VerticalSpacer.svelte";
    import { log } from "$lib/utils/logger";
    import { onMount, type Component } from "svelte";
    import { getSettingsProvider, getToastProvider } from "$lib/context";
    import { Save, RotateCcw, RefreshCw, TriangleAlert } from "@lucide/svelte";
    import Button from "$lib/components/ui/input/Button.svelte";
    import { fade, slide } from "svelte/transition";
    import Tabs from "$lib/components/ui/Tabs.svelte";
    import {
        createInputInstances,
        createSettingsSections,
        tabs,
        type SettingInputInstance,
        type TabId,
    } from "./settings-sections";

    const settings = getSettingsProvider();
    const toast = getToastProvider();
    const sections = createSettingsSections(settings);
    let activeTab = $state<TabId>("dashboard");

    const inputInstances = $state(createInputInstances(sections));

    let hasChanges = $state(false);
    let saving = $state(false);
    let inputsDisabled = $state(true);

    onMount(async () => {
        await Promise.all([settings.proxySettings.reload(), settings.dashboardSettings.reload()]);
        await resetInputs();
        inputsDisabled = false;
    });

    function allInputInstances() {
        return Object.values(inputInstances)
            .flat(2)
            .filter((input): input is SettingInputInstance => input != null);
    }

    function onChange(_different: boolean) {
        hasChanges = allInputInstances().some((i) => i.hasDiverged());
    }

    async function commitChanges() {
        saving = true;
        try {
            await Promise.all(allInputInstances().map((i) => i.commit()));

            toast.success("Settings saved successfully.");
            await settings.proxySettings.reload();
            await resetInputs();
        } catch (e) {
            log.error("Failed to save settings:", e);
            toast.error(e instanceof Error ? e.message : String(e));
        } finally {
            saving = false;
        }
    }

    async function resetInputs() {
        await Promise.all(allInputInstances().map((i) => i.reset()));
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
        grid-template-columns: repeat(auto-fill, minmax(min(100%, 350px), 1fr));
        gap: 1.5rem 3rem;
        width: 100%;
    }

    .action-bar {
        position: fixed;
        bottom: 2rem;
        left: 50%;
        transform: translateX(-50%);
        width: calc(100% - 2rem);
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
        gap: 1rem;
    }

    @media (max-width: 768px) {
        .action-content {
            flex-direction: column;
            text-align: center;
        }

        .buttons {
            width: 100%;
            justify-content: center;
        }

        .action-bar {
            bottom: 1rem;
            padding: 0.75rem 1rem;
        }
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
