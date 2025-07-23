<script lang="ts">
    import { getTimingMetrics, TimingMetrics } from "$lib/api/metrics/timing-metrics";
    import { formatTimeSinceDate } from "$lib/utils/dates";
    import Widget from "./Widget.svelte";

    let metrics: TimingMetrics | null = $state(null);
    let error: any | null = $state(null);

    async function fetchMetrics() {
        console.log("Fetching system metrics...");
        try {
            metrics = await getTimingMetrics();
        } catch (err) {
            error = err;
        }
    }
</script>

<Widget title="System Metrics" onPoll={fetchMetrics}>
    {#if metrics === null && error === null}
        <p>Loading...</p>
    {:else if error}
        <p class="error"><strong>Error fetching metrics:</strong> {error.message}</p>
    {:else if metrics}
        <div class="metrics">
            <p><strong>Uptime:</strong> {formatTimeSinceDate(metrics.startTime)}</p>
        </div>
    {:else}
        <p>Loading metrics...</p>
    {/if}
</Widget>
