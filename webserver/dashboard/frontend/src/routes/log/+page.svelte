<script lang="ts">
    import { apiGetTextStream } from "$lib/api/api-object";
    import PageTitle from "$lib/components/ui/PageTitle.svelte";
    import TextViewer from "$lib/components/ui/TextViewer.svelte";
    import { onDestroy, onMount } from "svelte";

    let textViewer: TextViewer;
    let logEvent: EventSource | undefined;

    onMount(async () => {
        let textStream = await apiGetTextStream("/log");

        const reader = textStream.getReader();
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            textViewer.appendText(value);
        }

        logEvent = new EventSource("/api/log/stream");
        logEvent.onmessage = (event) => {
            textViewer.appendText(event.data + "\n");
        };
    });

    onDestroy(() => {
        logEvent?.close();
    });
</script>

<PageTitle --pagetitle-margin-bottom="1.5rem">Log</PageTitle>
<TextViewer bind:this={textViewer} --viewer-max-height="90%" />
