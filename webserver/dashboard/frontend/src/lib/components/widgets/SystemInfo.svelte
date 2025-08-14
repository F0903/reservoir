<script lang="ts">
    import { formatTimeSinceDate } from "$lib/utils/dates";
    import ErrorBox from "../ui/ErrorBox.svelte";
    import type { MetricsProvider } from "$lib/providers/metrics.svelte";
    import PolledWidget from "./base/PolledWidget.svelte";
    import { getContext } from "svelte";

    let metrics = getContext("metrics") as MetricsProvider;

    let currentUptime: string = $state("");

    function updateUptime() {
        if (!metrics) {
            return;
        }

        const startTime = metrics.data.timing.startTime;
        currentUptime = formatTimeSinceDate(startTime);
    }
</script>

<PolledWidget title="System Metrics" pollFn={updateUptime} pollInterval={1000}>
    {#if metrics.state.initializing}
        <p>Loading...</p>
    {:else if metrics.state.error}
        <ErrorBox><p>{metrics.state.error}</p></ErrorBox>
    {:else}
        <div class="metrics">
            <p><strong>Uptime:</strong> {currentUptime}</p>
        </div>
    {/if}
</PolledWidget>
