<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Widget from "./base/Widget.svelte";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";

    const metrics = getContext("metrics") as MetricsProvider;
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
                <div class="cards">
                    <MetricCard
                        label="Total Requests"
                        value={totalRequests.toLocaleString()}
                        --metric-label-color="var(--secondary-600)"
                        --metric-value-color="var(--tertiary-400)"
                    />
                    <MetricCard label="HTTP Share" value={`${httpShare.toFixed(1)}%`} />
                    <MetricCard label="HTTPS Share" value={`${httpsShare.toFixed(1)}%`} />
                </div>

                <div class="chart-container">
                    <Chart
                        type="bar"
                        data={{
                            labels: ["Requests"],
                            datasets: [
                                {
                                    label: "HTTP",
                                    data: [httpRequests],
                                },
                                {
                                    label: "HTTPS",
                                    data: [httpsRequests],
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

    .cards {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
        gap: 0.75rem;
    }

    .chart-container {
        flex: 1;
        min-height: 0;
    }
</style>
