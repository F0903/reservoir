<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metrics.svelte";
    import { getContext } from "svelte";
    import ErrorBox from "../ui/ErrorBox.svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import { log } from "$lib/utils/logger";

    let metrics = getContext("metrics") as MetricsProvider;

    let chart: Chart | undefined = $state();

    let totalCacheRequests = $derived(
        metrics.data.cache.cacheHits + metrics.data.cache.cacheMisses,
    );
    let hitRate = $derived(
        totalCacheRequests > 0 ? (metrics.data.cache.cacheHits / totalCacheRequests) * 100 : 0,
    );

    let chartData = $derived({
        labels: ["Cache Operations"],
        datasets: [
            {
                label: "Hits",
                data: [metrics.data.cache.cacheHits],
                backgroundColor: "var(--secondary-400)",
            },
            {
                label: "Misses",
                data: [metrics.data.cache.cacheMisses],
                backgroundColor: "var(--tertiary-400)",
            },
            {
                label: "Errors",
                data: [metrics.data.cache.cacheErrors],
                backgroundColor: "var(--primary-200)",
            },
        ],
    });
</script>

<Widget title="Cache Efficiency">
    {#if metrics.state.initializing}
        <p>Loading...</p>
    {:else if metrics.state.error}
        <ErrorBox><p>{metrics.state.error}</p></ErrorBox>
    {:else}
        <div class="efficiency-display">
            <MetricCard
                label=" Hit Rate"
                value={hitRate.toFixed(1) + "%"}
                --metric-label-size="0.875rem"
                --metric-value-size="2rem"
                --metric-label-color="var(--secondary-600)"
                --metric-value-color="var(--tertiary-400)"
            />

            <div class="efficiency-chart">
                <Chart
                    bind:this={chart}
                    type="bar"
                    data={chartData}
                    options={{
                        scales: {
                            x: { stacked: true },
                            y: { stacked: true },
                        },
                    }}
                ></Chart>
            </div>
        </div>
    {/if}
</Widget>

<style>
    .efficiency-display {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    .efficiency-chart {
        height: 100%;
    }
</style>
