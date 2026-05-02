<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Widget from "./base/Widget.svelte";
    import { getMetricsProvider } from "$lib/context";

    const metrics = getMetricsProvider();
</script>

<Widget title="Request Volume">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            {@const proxiedRequests = data.requests.http_proxy_requests}
            {@const connectTunnels = data.requests.https_proxy_requests}
            {@const upstreamRequests = data.requests.upstream_requests}

            <div class="volume-wrapper">
                <div class="metric-cards-container">
                    <MetricCard
                        label="Proxied Requests"
                        value={proxiedRequests.toLocaleString()}
                        --metric-value-color="var(--tertiary-400)"
                    />
                    <MetricCard label="CONNECT Tunnels" value={connectTunnels.toLocaleString()} />
                    <MetricCard
                        label="Upstream Requests"
                        value={upstreamRequests.toLocaleString()}
                    />
                </div>

                <div class="chart-container hide-on-mobile">
                    <Chart
                        type="bar"
                        data={{
                            labels: ["Proxied", "CONNECT", "Upstream"],
                            datasets: [
                                {
                                    label: "Count",
                                    data: [proxiedRequests, connectTunnels, upstreamRequests],
                                    backgroundColor: "var(--secondary-400)",
                                },
                            ],
                        }}
                        options={{
                            indexAxis: "y",
                            scales: {
                                x: {
                                    title: {
                                        display: true,
                                        text: "Count",
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
    .volume-wrapper {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        height: 100%;
    }

    .metric-cards-container {
        display: flex;
        flex-direction: row;
        gap: 0.75rem;
    }

    .chart-container {
        flex: 1;
        min-height: 0;
    }

    @media (max-width: 768px) {
        .metric-cards-container {
            flex-direction: column;
            height: 100%;
        }
    }
</style>
