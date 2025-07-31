<script lang="ts">
    import Chart from "$lib/charts/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metrics.svelte";
    import ErrorBox from "../ui/ErrorBox.svelte";
    import Widget from "./base/Widget.svelte";

    let { metrics }: { metrics: MetricsProvider } = $props();
</script>

<Widget title="Cache Performance">
    {#if metrics.state.initializing}
        <p>Loading...</p>
    {:else if metrics.state.error}
        <ErrorBox><p>{metrics.state.error}</p></ErrorBox>
    {:else}
        <Chart
            type="doughnut"
            data={{
                labels: ["Cache Hits", "Cache Misses", "Cache Errors"],
                datasets: [
                    {
                        data: [
                            metrics.data.cache.cacheHits,
                            metrics.data.cache.cacheMisses,
                            metrics.data.cache.cacheErrors,
                        ],
                    },
                ],
            }}
            options={{
                plugins: {
                    legend: {
                        position: "bottom",
                    },
                },
            }}
        ></Chart>
    {/if}
</Widget>
