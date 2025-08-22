<script lang="ts">
    import { apiGetTextStream } from "$lib/api/api-object";
    import { onMount } from "svelte";

    let logContainer: HTMLPreElement;

    onMount(async () => {
        let textStream = await apiGetTextStream("/log");
        const reader = textStream.getReader();
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            logContainer.textContent += value;
        }
    });

    //TODO: implement/use the SSE stream and append new logs
    //TODO: make a nice container/UI to contain the log
</script>

<h1>Log Stream</h1>
<pre bind:this={logContainer}></pre>
