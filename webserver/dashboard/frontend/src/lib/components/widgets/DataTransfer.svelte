<script lang="ts">
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { formatBytesToLargest } from "$lib/utils/bytestring";
    import MetricCard from "./utils/MetricCard.svelte";

    const metrics = getContext("metrics") as MetricsProvider;
</script>

<Widget title="Data Transfer">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            {@const totalRequests =
                data.requests.http_proxy_requests + data.requests.https_proxy_requests}
            {@const avgBytesPerRequest = data.requests.bytes_served / totalRequests}
            <div class="cards">
                <MetricCard
                    label="Bytes Served"
                    value={formatBytesToLargest(data.requests.bytes_served)}
                    --metric-padding=".7rem"
                    --metric-label-color="var(--secondary-600)"
                    --metric-value-color="var(--tertiary-400)"
                    --metric-height="100%"
                />
                <MetricCard
                    label="Bytes Fetched"
                    value={formatBytesToLargest(data.requests.bytes_fetched)}
                    --metric-padding=".7rem"
                    --metric-label-color="var(--secondary-600)"
                    --metric-value-color="var(--tertiary-400)"
                    --metric-height="100%"
                />
                <MetricCard
                    label="Bandwidth Saved"
                    value={formatBytesToLargest(
                        data.requests.bytes_served - data.requests.bytes_fetched,
                    )}
                    --metric-padding=".7rem"
                    --metric-label-color="var(--secondary-600)"
                    --metric-value-color="var(--tertiary-400)"
                    --metric-height="100%"
                />
                <MetricCard
                    label="Total Requests"
                    value={totalRequests.toLocaleString()}
                    --metric-padding=".7rem"
                    --metric-height="100%"
                />
                <MetricCard
                    label="Avg per Request"
                    value={formatBytesToLargest(avgBytesPerRequest ? avgBytesPerRequest : 0)}
                    --metric-padding=".7rem"
                    --metric-height="100%"
                />
            </div>
        {/snippet}
    </Loadable>
</Widget>

<style>
    .cards {
        display: flex;
        flex-direction: column;
        gap: 5px;
        height: 100%;
        min-height: 0;
    }
</style>
