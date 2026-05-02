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
                data.cache.cache_request_hits +
                data.cache.cache_request_revalidations +
                data.cache.cache_request_stales +
                data.cache.cache_request_misses}
            {@const hitRate =
                totalCacheRequests > 0
                    ? ((data.cache.cache_request_hits +
                          data.cache.cache_request_revalidations +
                          data.cache.cache_request_stales) /
                          totalCacheRequests) *
                      100
                    : 0}
            {@const missRate =
                totalCacheRequests > 0
                    ? (data.cache.cache_request_misses / totalCacheRequests) * 100
                    : 0}
            {@const revalidationRate =
                totalCacheRequests > 0
                    ? (data.cache.cache_request_revalidations / totalCacheRequests) * 100
                    : 0}
            <div class="efficiency-display">
                <div class="metric-cards-container">
                    <MetricCard
                        label="Served From Cache"
                        value={hitRate.toFixed(1) + "%"}
                        --metric-value-color="var(--success-color)"
                    />
                    <MetricCard label="Miss Rate" value={missRate.toFixed(1) + "%"} />
                    <MetricCard
                        label="Revalidation Rate"
                        value={revalidationRate.toFixed(1) + "%"}
                    />
                </div>

                <div class="efficiency-chart">
                    <Chart
                        type="bar"
                        data={{
                            labels: ["Proxy Cache Outcomes"],
                            datasets: [
                                {
                                    label: "Fresh Hits",
                                    data: [data.cache.cache_request_hits],
                                    backgroundColor: "var(--success-color)",
                                },
                                {
                                    label: "Revalidations",
                                    data: [data.cache.cache_request_revalidations],
                                    backgroundColor: "var(--secondary-400)",
                                },
                                {
                                    label: "Stale",
                                    data: [data.cache.cache_request_stales],
                                    backgroundColor: "var(--tertiary-400)",
                                },
                                {
                                    label: "Misses",
                                    data: [data.cache.cache_request_misses],
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

    .metric-cards-container {
        display: flex;
        flex-direction: row;
        gap: 0.75rem;
    }

    .efficiency-chart {
        flex: 1;
        min-height: 0;
    }

    @media (max-width: 768px) {
        .efficiency-chart {
            display: none;
        }

        .metric-cards-container {
            flex-direction: column;
            height: 100%;
        }
    }
</style>
