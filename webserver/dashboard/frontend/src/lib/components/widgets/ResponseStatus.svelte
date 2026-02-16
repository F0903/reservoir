<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import Widget from "./base/Widget.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { getMetricsProvider } from "$lib/context";

    const metrics = getMetricsProvider();
</script>

<Widget title="Response Status">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            <Chart
                type="bar"
                data={{
                    labels: ["2xx", "4xx", "5xx"],
                    datasets: [
                        {
                            label: "Request Status Codes",
                            data: [
                                data.requests.status_ok_responses,
                                data.requests.status_client_error_responses,
                                data.requests.status_server_error_responses,
                            ],
                            backgroundColor: [
                                "var(--success-color)", // 2xx
                                "var(--tertiary-400)", // 4xx
                                "var(--error-color)", // 5xx
                            ],
                        },
                    ],
                }}
            />
        {/snippet}
    </Loadable>
</Widget>
