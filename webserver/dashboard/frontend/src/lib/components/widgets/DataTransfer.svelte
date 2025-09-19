<script lang="ts">
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Card from "../ui/Card.svelte";
    import Widget from "./base/Widget.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { formatBytesToLargest } from "$lib/utils/bytestring";

    let metrics = getContext("metrics") as MetricsProvider;
</script>

<Widget title="Data Transfer">
    <Loadable loadable={metrics}>
        <Card --card-background="var(--primary-600)" --card-padding="1rem">
            <div class="primary-metric">
                <div class="primary-metric-value">
                    {formatBytesToLargest(metrics.data.requests.bytesServed)}
                </div>
                <div class="primary-metric-label label">Total Bytes Served</div>
            </div>

            <div class="secondary-metrics">
                <div class="secondary-metric">
                    <span class="secondary-metric-value"
                        >{(
                            metrics.data.requests.httpProxyRequests +
                            metrics.data.requests.httpsProxyRequests
                        ).toLocaleString()}</span
                    >
                    <span class="secondary-metric-label label">Total Requests</span>
                </div>

                {#if metrics.data.requests.httpProxyRequests + metrics.data.requests.httpsProxyRequests > 0}
                    <div class="secondary-metric">
                        <span class="secondary-metric-value"
                            >{formatBytesToLargest(
                                metrics.data.requests.bytesServed /
                                    (metrics.data.requests.httpProxyRequests +
                                        metrics.data.requests.httpsProxyRequests),
                            )}</span
                        >
                        <span class="secondary-metric-label label">Avg per Request</span>
                    </div>
                {/if}
            </div>
        </Card>
    </Loadable>
</Widget>

<style>
    .label {
        color: var(--primary-200);
    }

    .primary-metric {
        margin-bottom: 1.5rem;
        text-align: center;
    }

    .primary-metric-value {
        font-size: 2rem;
        font-weight: bold;
        color: var(--tertiary-400);
        margin-bottom: 0.5rem;
    }

    .primary-metric-label {
        font-size: 1rem;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .secondary-metrics {
        display: flex;
        justify-content: space-around;
        gap: 1rem;
    }

    .secondary-metric {
        display: flex;
        flex-direction: column;
        align-items: center;
    }

    .secondary-metric-value {
        font-size: 1.25rem;
        font-weight: bold;
        margin-bottom: 0.25rem;
        color: var(--secondary-400);
    }

    .secondary-metric-label {
        font-size: 0.75rem;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }
</style>
