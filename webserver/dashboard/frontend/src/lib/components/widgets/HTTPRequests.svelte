<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import Loadable from "../ui/Loadable.svelte";

    const metrics = getContext("metrics") as MetricsProvider;
</script>

<Widget title="HTTP/HTTPS Requests">
    <Loadable state={metrics.data} error={metrics.error}>
        <Chart
            type="doughnut"
            data={{
                labels: ["HTTP Proxy Requests", "HTTPS Proxy Requests"],
                datasets: [
                    {
                        data: [
                            metrics.data!.requests.http_proxy_requests,
                            metrics.data!.requests.https_proxy_requests,
                        ],
                    },
                ],
            }}
        ></Chart>
    </Loadable>
</Widget>
