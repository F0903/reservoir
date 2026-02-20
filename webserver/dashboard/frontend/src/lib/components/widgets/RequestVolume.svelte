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
            {@const httpRequests = data.requests.http_proxy_requests}
            {@const httpsRequests = data.requests.https_proxy_requests}
            {@const totalRequests = httpRequests + httpsRequests}
            {@const httpShare = totalRequests > 0 ? (httpRequests / totalRequests) * 100 : 0}
            {@const httpsShare = totalRequests > 0 ? (httpsRequests / totalRequests) * 100 : 0}

            <div class="volume-wrapper">
                <div class="metric-cards-container">
                    <MetricCard
                        label="Total Requests"
                        value={totalRequests.toLocaleString()}
                        --metric-value-color="var(--tertiary-400)"
                    />
                    <MetricCard label="HTTP Share" value={`${httpShare.toFixed(1)}%`} />
                    <MetricCard label="HTTPS Share" value={`${httpsShare.toFixed(1)}%`} />
                </div>

                <div class="chart-container hide-on-mobile">
                    <Chart
                        type="bar"
                        data={{
                            labels: ["Requests"],
                            datasets: [
                                {
                                    label: "HTTP",
                                    data: [httpRequests],
                                    backgroundColor: "var(--secondary-400)",
                                },
                                {
                                    label: "HTTPS",
                                    data: [httpsRequests],
                                    backgroundColor: "var(--secondary-300)",
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
                                        text: "Count",
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
