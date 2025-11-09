<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";

    const metrics = getContext("metrics") as MetricsProvider;
</script>

<Widget title="Cache Efficiency">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            {@const totalCacheRequests =
                data.cache.cache_hits + data.cache.cache_misses + data.cache.cache_errors}
            {@const hitRate =
                totalCacheRequests > 0 ? (data.cache.cache_hits / totalCacheRequests) * 100 : 0}
            <div class="efficiency-display">
                <div class="metric-card-container">
                    <MetricCard
                        label="Hit Rate"
                        value={hitRate.toFixed(1) + "%"}
                        --metric-label-color="var(--secondary-600)"
                        --metric-value-color="var(--tertiary-400)"
                    />
                </div>

                <div class="efficiency-chart">
                    <Chart
                        type="bar"
                        data={{
                            labels: ["Cache Operations"],
                            datasets: [
                                {
                                    label: "Hits",
                                    data: [data.cache.cache_hits],
                                },
                                {
                                    label: "Misses",
                                    data: [data.cache.cache_misses],
                                },
                                {
                                    label: "Errors",
                                    data: [data.cache.cache_errors],
                                },
                            ],
                        }}
                        options={{
                            scales: {
                                x: { stacked: true },
                                y: { stacked: true },
                            },
                        }}
                    ></Chart>
                </div>
            </div>
        {/snippet}
    </Loadable>
</Widget>

<style>
    .efficiency-display {
        display: flex;
        gap: 1rem;
        flex-direction: column;
        height: 100%;
    }

    .efficiency-chart {
        min-height: 0;
    }
</style>
