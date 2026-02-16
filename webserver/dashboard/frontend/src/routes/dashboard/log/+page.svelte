<script lang="ts">
    import { getLogStream } from "$lib/api/objects/log/log";
    import PageTitle from "$lib/components/ui/PageTitle.svelte";
    import TextViewer from "$lib/components/ui/TextViewer.svelte";
    import Button from "$lib/components/ui/input/Button.svelte";
    import Toggle from "$lib/components/ui/input/Toggle.svelte";
    import { log } from "$lib/utils/logger";
    import { onDestroy, onMount } from "svelte";

    let textViewer: TextViewer;
    let logEvent: EventSource | undefined = $state(undefined);

    let autoScroll = $state(true);
    let paused = $state(false);

    onMount(async () => {
        const fetchHistory = async () => {
            let textStream = await getLogStream();
            const reader = textStream.getReader();

            while (true) {
                const { done, value } = await reader.read();
                if (done) break;
                await textViewer.appendText(value);
            }
        };

        await fetchHistory();
        startLogListen();
    });

    onDestroy(() => {
        stopLogListen();
    });

    $effect(() => {
        log.debug("Effect triggered");
        if (paused) {
            stopLogListen();
        } else {
            startLogListen();
        }
    });

    function stopLogListen() {
        if (!logEvent) return;

        log.debug("Stopping log listen");
        logEvent.close();
        logEvent = undefined;
    }

    function startLogListen() {
        if (logEvent) return;

        log.debug("Starting log listen");
        logEvent = new EventSource("/api/log/stream");
        logEvent.onmessage = (event) => {
            if (!paused) {
                textViewer.appendText(event.data + "\n");
            }
        };
    }

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
            <Button onClick={clearLog}>Clear</Button>
        </div>
    </div>
    <TextViewer
        bind:this={textViewer}
        bind:autoScroll
        syntaxHighlighting="slog"
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
