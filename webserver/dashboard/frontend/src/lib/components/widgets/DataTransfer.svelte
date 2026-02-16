<script lang="ts">
    import Widget from "./base/Widget.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { formatBytesToLargest } from "$lib/utils/bytestring";
    import MetricCard from "./utils/MetricCard.svelte";
    import { getMetricsProvider } from "$lib/context";

    const metrics = getMetricsProvider();
</script>

<Widget title="Data Transfer">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            {@const totalRequests =
                data.requests.http_proxy_requests + data.requests.https_proxy_requests}
            {@const avgBytesPerRequest =
                totalRequests > 0 ? data.requests.bytes_served / totalRequests : 0}
            {@const startTime = data.system?.start_time ? new Date(data.system.start_time) : null}
            {@const uptimeSeconds = startTime
                ? Math.max((Date.now() - startTime.getTime()) / 1000, 1)
                : 0}
            {@const servedPerSecond =
                uptimeSeconds > 0 ? data.requests.bytes_served / uptimeSeconds : 0}
            {@const fetchedPerSecond =
                uptimeSeconds > 0 ? data.requests.bytes_fetched / uptimeSeconds : 0}
            {@const requestsPerSecond = uptimeSeconds > 0 ? totalRequests / uptimeSeconds : 0}

            <div class="cards">
                <MetricCard
                    label="Bytes Served"
                    value={formatBytesToLargest(data.requests.bytes_served)}
                />
                <MetricCard
                    label="Bytes Fetched"
                    value={formatBytesToLargest(data.requests.bytes_fetched)}
                />
                <MetricCard
                    label="Bandwidth Saved"
                    value={formatBytesToLargest(
                        data.requests.bytes_served - data.requests.bytes_fetched,
                    )}
                    --metric-value-color="var(--success-color)"
                />
                <MetricCard
                    label="Served Throughput"
                    value={`${formatBytesToLargest(servedPerSecond)}/s`}
                />
                <MetricCard
                    label="Fetched Throughput"
                    value={`${formatBytesToLargest(fetchedPerSecond)}/s`}
                />
                <MetricCard label="Requests / s" value={requestsPerSecond.toFixed(2)} />
                <MetricCard
                    label="Total Requests"
                    value={totalRequests.toLocaleString()}
                    --metric-value-color="var(--tertiary-400)"
                />
                <MetricCard
                    label="Avg per Request"
                    value={formatBytesToLargest(avgBytesPerRequest)}
                />
            </div>
        {/snippet}
    </Loadable>
</Widget>

<style>
    .cards {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 0.75rem;
        height: 100%;
        min-height: 0;
    }
</style>
