<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import Loadable from "../ui/Loadable.svelte";

    const metrics = getContext("metrics") as MetricsProvider;
</script>

<Widget title="Cache Performance">
    <Loadable state={metrics.data} loadable={metrics}>
        <Chart
            type="doughnut"
            data={{
                labels: ["Cache Hits", "Cache Misses", "Cache Errors"],
                datasets: [
                    {
                        data: [
                            metrics.data!.cache.cache_hits,
                            metrics.data!.cache.cache_misses,
                            metrics.data!.cache.cache_errors,
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
