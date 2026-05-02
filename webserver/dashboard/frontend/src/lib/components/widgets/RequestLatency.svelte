<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import Loadable from "$lib/components/ui/Loadable.svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import { getMetricsProvider } from "$lib/context";

    const metrics = getMetricsProvider();

    const nanosToMillis = (value: number) => value / 1e6;
</script>

<Widget title="Request Latency">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            {@const totalProxiedRequests = data.requests.http_proxy_requests}
            {@const totalClientResponses = data.requests.client_responses}
            {@const totalUpstreamRequests = data.requests.upstream_requests}
            {@const avgEndToEndLatencyMs =
                totalProxiedRequests > 0
                    ? nanosToMillis(data.requests.client_request_latency / totalProxiedRequests)
                    : 0}
            {@const avgClientResponseLatencyMs =
                totalClientResponses > 0
                    ? nanosToMillis(data.requests.client_response_latency / totalClientResponses)
                    : 0}
            {@const avgUpstreamLatencyMs =
                totalUpstreamRequests > 0
                    ? nanosToMillis(data.requests.upstream_request_latency / totalUpstreamRequests)
                    : 0}

            <div class="latency-wrapper">
                <div class="cards">
                    <MetricCard
                        label="End-to-End Request"
                        value={`${avgEndToEndLatencyMs.toFixed(2)} ms`}
                        --metric-value-color="var(--tertiary-400)"
                    />
                    <MetricCard
                        label="Reservoir to Client"
                        value={`${avgClientResponseLatencyMs.toFixed(2)} ms`}
                        --metric-value-color="var(--tertiary-400)"
                    />
                    <MetricCard
                        label="Upstream Fetch"
                        value={`${avgUpstreamLatencyMs.toFixed(2)} ms`}
                        --metric-value-color="var(--tertiary-600)"
                    />
                    <MetricCard
                        label="Upstream Calls"
                        value={totalUpstreamRequests.toLocaleString()}
                    />
                </div>

                <div class="chart-container hide-on-mobile">
                    <Chart
                        type="bar"
                        data={{
                            labels: ["Average Latency"],
                            datasets: [
                                {
                                    label: "End-to-End",
                                    data: [avgEndToEndLatencyMs],
                                    backgroundColor: "var(--tertiary-400)",
                                },
                                {
                                    label: "Reservoir to Client",
                                    data: [avgClientResponseLatencyMs],
                                    backgroundColor: "var(--secondary-400)",
                                },
                                {
                                    label: "Upstream",
                                    data: [avgUpstreamLatencyMs],
                                    backgroundColor: "var(--tertiary-600)",
                                },
                            ],
                        }}
                        options={{
                            indexAxis: "y",
                            scales: {
                                x: {
                                    title: {
                                        display: true,
                                        text: "Milliseconds",
                                    },
                                },
                            },
                        }}
                    />
                </div>
            </div>
        {/snippet}
    </Loadable>
</Widget>

<style>
    .latency-wrapper {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        height: 100%;
    }

    .cards {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
        gap: 0.75rem;
    }

    .chart-container {
        flex: 1;
        min-height: 0;
    }

    @media (max-width: 768px) {
        .hide-on-mobile {
            display: none;
        }

        .cards {
            grid-template-columns: 1fr 1fr;
            flex: 1;
            min-height: 0;
            height: 100%;
        }
    }
</style>
