<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";

    const metrics = getContext("metrics") as MetricsProvider;
</script>

<Widget title="Request Coalescing">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            {@const totalCoalescedRequests = data.requests.coalesced_requests}
            {@const totalRequests = totalCoalescedRequests + data.requests.non_coalesced_requests}
            {@const coalescingRate =
                totalRequests > 0 ? (totalCoalescedRequests / totalRequests) * 100 : 0}
            {@const coalescedHits = data.requests.coalesced_cache_hits}
            {@const coalescedRevalidations = data.requests.coalesced_cache_revalidations}
            {@const coalescedMisses = data.requests.coalesced_cache_misses}
            {@const resolvedCoalescedRequests =
                coalescedHits + coalescedRevalidations + coalescedMisses}
            {@const coalescedCacheHitRate =
                resolvedCoalescedRequests > 0
                    ? (coalescedHits / resolvedCoalescedRequests) * 100
                    : 0}
            {@const coalescedRevalidationRate =
                resolvedCoalescedRequests > 0
                    ? (coalescedRevalidations / resolvedCoalescedRequests) * 100
                    : 0}

            <div class="coalescing-display">
                <div class="metrics-row">
                    <MetricCard
                        label="Coalescing Rate"
                        value={coalescingRate.toFixed(1) + "%"}
                        --metric-label-color="var(--secondary-600)"
                        --metric-value-color="var(--tertiary-400)"
                    />
                    <MetricCard
                        label="Coalesced Requests"
                        value={totalCoalescedRequests.toString()}
                        --metric-label-color="var(--secondary-600)"
                        --metric-value-color="var(--secondary-400)"
                    />
                    <MetricCard
                        label="Cache Hit Rate"
                        value={coalescedCacheHitRate.toFixed(1) + "%"}
                        --metric-label-color="var(--secondary-600)"
                        --metric-value-color="var(--primary-400)"
                    />
                    <MetricCard
                        label="Revalidation Rate"
                        value={coalescedRevalidationRate.toFixed(1) + "%"}
                        --metric-label-color="var(--secondary-600)"
                        --metric-value-color="var(--primary-500)"
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
                                    data: [coalescedHits],
                                },
                                {
                                    label: "Revalidations",
                                    data: [coalescedRevalidations],
                                },
                                {
                                    label: "Cache Misses",
                                    data: [coalescedMisses],
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
    .coalescing-display {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        height: 100%;
    }

    .metrics-row {
        display: flex;
        gap: 1rem;
    }

    .coalescing-chart {
        flex-grow: 1;
        min-height: 0;
        height: 100%;
    }
</style>
