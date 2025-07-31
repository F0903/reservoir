<script lang="ts">
    import { CacheMetrics, getCacheMetrics } from "$lib/api/objects/metrics/cache-metrics";
    import ErrorBox from "../ui/ErrorBox.svelte";
    import Widget from "./Widget.svelte";

    let metrics: CacheMetrics | null = $state(null);
    let error: any | null = $state(null);

    async function fetchMetrics() {
        console.log("Fetching cache metrics...");
        try {
            metrics = await getCacheMetrics();
        } catch (err) {
            error = err;
        }
    }
</script>

<Widget title="Cache Metrics" onPoll={fetchMetrics}>
    {#if metrics === null && error === null}
        <p>Loading...</p>
    {:else if error}
        <ErrorBox><p>{error.message}</p></ErrorBox>
    {:else if metrics}
        <div class="metrics">
            <p><strong>Cache Hits:</strong> {metrics.cacheHits}</p>
            <p><strong>Cache Misses:</strong> {metrics.cacheMisses}</p>
            <p><strong>Cache Errors:</strong> {metrics.cacheErrors}</p>
            <p><strong>Cache Entries:</strong> {metrics.cacheEntries}</p>
            <p><strong>Bytes Cached:</strong> {metrics.bytesCached}</p>
            <p><strong>Cleanup Runs:</strong> {metrics.cleanupRuns}</p>
            <p><strong>Bytes Cleaned:</strong> {metrics.bytesCleaned}</p>
            <p><strong>Cache Evictions:</strong> {metrics.cacheEvictions}</p>
        </div>
    {:else}
        <p>Loading metrics...</p>
    {/if}
</Widget>
