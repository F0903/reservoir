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
            {@const totalClientRequests =
                data.requests.http_proxy_requests + data.requests.https_proxy_requests}
            {@const totalUpstreamRequests = data.requests.upstream_requests}
            {@const avgClientLatencyMs =
                totalClientRequests > 0
                    ? nanosToMillis(data.requests.client_request_latency / totalClientRequests)
                    : 0}
            {@const avgUpstreamLatencyMs =
                totalUpstreamRequests > 0
                    ? nanosToMillis(data.requests.upstream_request_latency / totalUpstreamRequests)
                    : 0}
            {@const clientContribution =
                avgClientLatencyMs + avgUpstreamLatencyMs > 0
                    ? (avgClientLatencyMs / (avgClientLatencyMs + avgUpstreamLatencyMs)) * 100
                    : 0}

            <div class="latency-wrapper">
                <div class="cards">
                    <MetricCard
                        label="Client → Reservoir"
                        value={`${avgClientLatencyMs.toFixed(2)} ms`}
                        --metric-value-color="var(--tertiary-400)"
                    />
                    <MetricCard
                        label="Reservoir → Upstream"
                        value={`${avgUpstreamLatencyMs.toFixed(2)} ms`}
                        --metric-value-color="var(--tertiary-400)"
                    />
                    <div class="double-span">
                        <MetricCard
                            label="Client Share"
                            value={`${clientContribution.toFixed(0)}%`}
                        />
                    </div>
                </div>

                <div class="chart-container hide-on-mobile">
                    <Chart
                        type="bar"
                        data={{
                            labels: ["Average Latency"],
                            datasets: [
                                {
                                    label: "Client",
                                    data: [avgClientLatencyMs],
                                    backgroundColor: "var(--tertiary-400)",
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
                                    stacked: true,
                                    title: {
                                        display: true,
                                        text: "Milliseconds",
                                    },
                                },
                                y: {
                                    stacked: true,
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

    .double-span {
        grid-column: span 2;
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

        .double-span {
            grid-column: span 2;
            height: 100%;
            display: flex;
        }
    }
</style>
