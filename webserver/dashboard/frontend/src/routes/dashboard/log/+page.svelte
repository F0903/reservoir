<script lang="ts">
    import { apiGetTextStream } from "$lib/api/api-helpers";
    import PageTitle from "$lib/components/ui/PageTitle.svelte";
    import TextViewer from "$lib/components/ui/TextViewer.svelte";
    import Button from "$lib/components/ui/input/Button.svelte";
    import Toggle from "$lib/components/ui/input/Toggle.svelte";
    import { onDestroy, onMount } from "svelte";

    let textViewer: TextViewer;
    let logEvent: EventSource | undefined;

    let autoScroll = $state(true);
    let paused = $state(false);

    onMount(async () => {
        let textStream = await apiGetTextStream("/log");

        const reader = textStream.getReader();
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            if (!paused) {
                await textViewer.appendText(value);
            }
        }

        logEvent = new EventSource("/api/log/stream");
        logEvent.onmessage = (event) => {
            if (!paused) {
                textViewer.appendText(event.data + "\n");
            }
        };
    });

    onDestroy(() => {
        logEvent?.close();
    });

    function clearLog() {
        textViewer.clear();
    }
</script>

<main class="page-container">
    <div class="header">
        <PageTitle --pagetitle-margin-bottom="0">Log</PageTitle>
        <div class="actions">
            <div class="toggles">
                <Toggle label="Auto-scroll" bind:value={autoScroll} --toggle-width="auto" />
                <Toggle label="Paused" bind:value={paused} --toggle-width="auto" />
            </div>
            <Button
                onClick={clearLog}
                --btn-background-color="var(--primary-450)"
                --btn-text-color="var(--text-400)"
            >
                Clear
            </Button>
        </div>
    </div>
    <TextViewer
        bind:this={textViewer}
        isLogViewer={true}
        {autoScroll}
        --viewer-max-height="calc(100% - 80px)"
    />
</main>

<style>
    .page-container {
        height: 100%;
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .actions {
        display: flex;
        align-items: center;
        gap: 2rem;
    }

    .toggles {
        display: flex;
        gap: 1.5rem;
    }

    :global(.actions .input-container) {
        margin: 0 !important;
    }
</style>
