<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import Loadable from "../ui/Loadable.svelte";

    let metrics = getContext("metrics") as MetricsProvider;

    let chart: Chart | undefined = $state();
</script>

<Widget title="Requests">
    <Loadable loadable={metrics}>
        <Chart
            bind:this={chart}
            type="doughnut"
            data={{
                labels: ["HTTP Proxy Requests", "HTTPS Proxy Requests"],
                datasets: [
                    {
                        data: [
                            metrics.data.requests.httpProxyRequests,
                            metrics.data.requests.httpsProxyRequests,
                        ],
                    },
                ],
            }}
        ></Chart>
    </Loadable>
</Widget>
