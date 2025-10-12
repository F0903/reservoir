<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";

    const metrics = getContext("metrics") as MetricsProvider;

    let totalCacheRequests = $derived(
        (metrics.data?.cache.cache_hits ?? 0) + (metrics.data?.cache.cache_misses ?? 0),
    );
    let hitRate = $derived(
        totalCacheRequests > 0
            ? ((metrics.data?.cache.cache_hits ?? 0) / totalCacheRequests) * 100
            : 0,
    );
</script>

<Widget title="Cache Efficiency">
    <Loadable state={metrics.data} error={metrics.error}>
        <div class="efficiency-display">
            <MetricCard
                label="Hit Rate"
                value={hitRate.toFixed(1) + "%"}
                --metric-label-size="0.875rem"
                --metric-value-size="2rem"
                --metric-label-color="var(--secondary-600)"
                --metric-value-color="var(--tertiary-400)"
            />

            <div class="efficiency-chart">
                <Chart
                    type="bar"
                    data={{
                        labels: ["Cache Operations"],
                        datasets: [
                            {
                                label: "Hits",
                                data: [metrics!.data!.cache.cache_hits],
                                backgroundColor: "var(--secondary-400)",
                            },
                            {
                                label: "Misses",
                                data: [metrics!.data!.cache.cache_misses],
                                backgroundColor: "var(--tertiary-400)",
                            },
                            {
                                label: "Errors",
                                data: [metrics!.data!.cache.cache_errors],
                                backgroundColor: "var(--primary-200)",
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
    </Loadable>
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
