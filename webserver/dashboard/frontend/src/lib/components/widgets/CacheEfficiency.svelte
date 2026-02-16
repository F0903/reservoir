<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { getMetricsProvider } from "$lib/context";

    const metrics = getMetricsProvider();
</script>

<Widget title="Cache Efficiency">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            {@const totalCacheRequests =
                data.cache.cache_hits + data.cache.cache_misses + data.cache.cache_errors}
            {@const hitRate =
                totalCacheRequests > 0 ? (data.cache.cache_hits / totalCacheRequests) * 100 : 0}
            {@const missRate =
                totalCacheRequests > 0 ? (data.cache.cache_misses / totalCacheRequests) * 100 : 0}
            {@const errorRate =
                totalCacheRequests > 0 ? (data.cache.cache_errors / totalCacheRequests) * 100 : 0}
            <div class="efficiency-display">
                <div class="metric-card-container">
                    <MetricCard
                        label="Hit Rate"
                        value={hitRate.toFixed(1) + "%"}
                        --metric-value-color="var(--success-color)"
                    />
                    <MetricCard label="Miss Rate" value={missRate.toFixed(1) + "%"} />
                    <MetricCard
                        label="Error Rate"
                        value={errorRate.toFixed(1) + "%"}
                        --metric-value-color="var(--log-error-color)"
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
                                    backgroundColor: "var(--success-color)",
                                },
                                {
                                    label: "Misses",
                                    data: [data.cache.cache_misses],
                                    backgroundColor: "var(--secondary-400)",
                                },
                                {
                                    label: "Errors",
                                    data: [data.cache.cache_errors],
                                    backgroundColor: "var(--error-color)",
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
        flex-direction: column;
        gap: 1rem;
        height: 100%;
        align-items: stretch;
    }

    .metric-card-container {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
        gap: 0.75rem;
    }

    .efficiency-chart {
        flex: 1;
        min-height: 0;
    }
</style>
