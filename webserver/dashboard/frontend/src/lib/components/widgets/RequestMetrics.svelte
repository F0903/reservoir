<script lang="ts">
    import { getRequestMetrics, RequestMetrics } from "$lib/api/metrics/request-metrics";
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
        <p class="error"><strong>Error fetching metrics:</strong> {error.message}</p>
    {:else if metrics}
        <div class="metrics">
            <p><strong>HTTP Proxy Requests:</strong> {metrics.httpProxyRequests}</p>
            <p><strong>HTTPS Proxy Requests:</strong> {metrics.httpsProxyRequests}</p>
            <p><strong>Bytes Served:</strong> {metrics.bytesServed}</p>
        </div>
    {:else}
        <p>Loading metrics...</p>
    {/if}
</Widget>
