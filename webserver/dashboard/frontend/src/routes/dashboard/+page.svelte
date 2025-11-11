<script lang="ts">
    import HTTPHTTPSRequests from "$lib/components/widgets/HTTPHTTPSRequests.svelte";
    import SystemInfo from "$lib/components/widgets/SystemInfo.svelte";
    import CacheLatency from "$lib/components/widgets/CacheLatency.svelte";
    import CacheRates from "$lib/components/widgets/CacheRates.svelte";
    import CacheStats from "$lib/components/widgets/CacheStats.svelte";
    import DataTransfer from "$lib/components/widgets/DataTransfer.svelte";
    import RequestCoalescing from "$lib/components/widgets/RequestCoalescing.svelte";
    import ResponseStatus from "$lib/components/widgets/ResponseStatus.svelte";
    import SquaredGrid from "$lib/components/layout/SquaredGrid.svelte";
    import { getContext, onMount } from "svelte";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";

    const metrics = getContext("metrics") as MetricsProvider;

    onMount(() => {
        metrics.refreshMetrics();
        metrics.startRefresh();

        return () => {
            metrics.stopRefresh();
        };
    });
</script>

<main class="dashboard">
    <SquaredGrid
        elements={[
            { Comp: CacheRates, span: { width: 2, height: 2 } },
            { Comp: CacheLatency, span: { width: 2, height: 2 } },
            { Comp: HTTPHTTPSRequests, span: { width: 2, height: 2 } },
            { Comp: ResponseStatus, span: { width: 2, height: 2 } },
            { Comp: RequestCoalescing, span: { width: 4, height: 3 } },
            { Comp: DataTransfer, span: { width: 2, height: 3 } },
            { Comp: SystemInfo, span: { width: 1, height: 2 } },
            { Comp: CacheStats, span: { width: 2, height: 3 } },
        ]}
    />
</main>

<style>
    main {
        height: 100%;
        width: 100%;
    }

    .dashboard {
        padding: 1.5rem;
    }

    @media (max-width: var(--mobile-cutoff)) {
        .dashboard {
            padding: 0;
        }
    }
</style>
