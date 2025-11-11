<script lang="ts">
    import Chart from "$lib/components/ui/Chart.svelte";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import Loadable from "../ui/Loadable.svelte";

    const metrics = getContext("metrics") as MetricsProvider;
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
                                "hsla(188, 34%, 43%)", // 2xx
                                "hsla(188, 34%, 30%)", // 4xx
                                "hsla(22, 70%, 44%)", // 5xx
                            ],
                        },
                    ],
                }}
            />
        {/snippet}
    </Loadable>
</Widget>
