<script lang="ts">
    import Chart from "$lib/charts/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metrics.svelte";
    import { getContext } from "svelte";
    import ErrorBox from "../ui/ErrorBox.svelte";
    import Widget from "./base/Widget.svelte";

    let metrics = getContext("metrics") as MetricsProvider;
</script>

<Widget title="Requests">
    {#if metrics.state.initializing}
        <p>Loading...</p>
    {:else if metrics.state.error}
        <ErrorBox><p>{metrics.state.error}</p></ErrorBox>
    {:else}
        <Chart
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
    {/if}
</Widget>
