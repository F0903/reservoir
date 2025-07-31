<script lang="ts">
    import { getRequestMetrics, RequestMetrics } from "$lib/api/objects/metrics/request-metrics";
    import Chart from "$lib/charts/Chart.svelte";
    import ErrorBox from "../ui/ErrorBox.svelte";
    import Widget from "./Widget.svelte";

    let metrics: RequestMetrics | null = $state(null);
    let error: any | null = $state(null);

    async function fetchMetrics() {
        console.log("Fetching request metrics...");
        try {
            metrics = await getRequestMetrics();
        } catch (err) {
            error = err;
        }
    }
</script>

<Widget title="Request Metrics" onPoll={fetchMetrics}>
    {#if metrics === null && error === null}
        <p>Loading...</p>
    {:else if error}
        <ErrorBox><p>{error.message}</p></ErrorBox>
    {:else if metrics}
        <Chart
            type="pie"
            data={{
                labels: ["HTTP Proxy Requests", "HTTPS Proxy Requests", "Bytes Served"],
                datasets: [
                    {
                        data: [
                            metrics.httpProxyRequests,
                            metrics.httpsProxyRequests,
                            metrics.bytesServed,
                        ],
                    },
                ],
            }}
        ></Chart>
    {:else}
        <p>Loading metrics...</p>
    {/if}
</Widget>
