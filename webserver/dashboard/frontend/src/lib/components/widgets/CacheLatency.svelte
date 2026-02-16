<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import Widget from "./base/Widget.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { getMetricsProvider } from "$lib/context";

    const metrics = getMetricsProvider();
</script>

<Widget title="Cache Latency">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            <!-- Convert nanoseconds to milliseconds -->
            {@const cacheHitLatencyMs =
                data.cache.cache_hit_latency / (data.cache.cache_hits || 1) / 1e6}
            {@const cacheMissLatencyMs =
                data.cache.cache_miss_latency / (data.cache.cache_misses || 1) / 1e6}

            <Chart
                type="bar"
                data={{
                    labels: ["Cache Hit", "Cache Miss"],
                    datasets: [
                        {
                            label: "Average Latency (ms)",
                            data: [cacheHitLatencyMs, cacheMissLatencyMs],
                            backgroundColor: [
                                "hsla(188, 34%, 43%)", // Cache Hit
                                "hsla(188, 34%, 30%)", // Cache Miss
                            ],
                        },
                    ],
                }}
                options={{
                    scales: {
                        y: {
                            type: "logarithmic",
                            ticks: {
                                callback: (tickValue: string | number) => {
                                    if (typeof tickValue === "number") {
                                        if (tickValue > 0 && Math.log10(tickValue) % 1 === 0) {
                                            return `${tickValue} ms`;
                                        }
                                    }
                                    return null;
                                },
                            },
                        },
                    },
                }}
            />
        {/snippet}
    </Loadable>
</Widget>
