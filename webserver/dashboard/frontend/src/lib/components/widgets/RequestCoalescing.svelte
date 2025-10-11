<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";

    let metrics = getContext("metrics") as MetricsProvider;

    let totalCoalescedRequests = $derived(
        (metrics.data?.requests.coalesced_requests ?? 0)
    );
    let totalRequests = $derived(
        totalCoalescedRequests + (metrics.data?.requests.non_coalesced_requests ?? 0)
    );
    let coalescingRate = $derived(
        totalRequests > 0 
            ? (totalCoalescedRequests / totalRequests) * 100
            : 0,
    );
    let coalescedCacheHitRate = $derived(
        totalCoalescedRequests > 0
            ? ((metrics.data?.requests.coalesced_cache_hits ?? 0) / totalCoalescedRequests) * 100
            : 0,
    );
</script>

<Widget title="Request Coalescing">
    <Loadable state={metrics.data} loadable={metrics}>
        <div class="coalescing-display">
            <div class="metrics-row">
                <MetricCard
                    label="Coalescing Rate"
                    value={coalescingRate.toFixed(1) + "%"}
                    --metric-label-size="0.875rem"
                    --metric-value-size="2rem"
                    --metric-label-color="var(--secondary-600)"
                    --metric-value-color="var(--tertiary-400)"
                />
                <MetricCard
                    label="Coalesced Requests"
                    value={totalCoalescedRequests.toString()}
                    --metric-label-size="0.875rem"
                    --metric-value-size="2rem"
                    --metric-label-color="var(--secondary-600)"
                    --metric-value-color="var(--secondary-400)"
                />
                <MetricCard
                    label="Cache Hit Rate"
                    value={coalescedCacheHitRate.toFixed(1) + "%"}
                    --metric-label-size="0.875rem"
                    --metric-value-size="2rem"
                    --metric-label-color="var(--secondary-600)"
                    --metric-value-color="var(--primary-400)"
                />
            </div>

            <div class="coalescing-chart">
                <Chart
                    type="bar"
                    data={{
                        labels: ["Coalesced Requests"],
                        datasets: [
                            {
                                label: "Cache Hits",
                                data: [metrics.data?.requests.coalesced_cache_hits ?? 0],
                                backgroundColor: "var(--secondary-400)",
                            },
                            {
                                label: "Cache Misses",
                                data: [metrics.data?.requests.coalesced_cache_misses ?? 0],
                                backgroundColor: "var(--tertiary-400)",
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
    .coalescing-display {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    .metrics-row {
        display: flex;
        gap: 1rem;
        flex-wrap: wrap;
    }

    .coalescing-chart {
        height: 100%;
    }
</style>
