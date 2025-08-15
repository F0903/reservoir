<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metrics.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import Loadable from "../ui/Loadable.svelte";

    let metrics = getContext("metrics") as MetricsProvider;
</script>

<Widget title="Cache Performance">
    <Loadable loadable={metrics}>
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
    </Loadable>
</Widget>
