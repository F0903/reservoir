<script lang="ts">
    import TotalRequests from "$lib/components/widgets/HTTPRequests.svelte";
    import SystemInfo from "$lib/components/widgets/SystemInfo.svelte";
    import CacheEfficiency from "$lib/components/widgets/CacheEfficiency.svelte";
    import CachePerformance from "$lib/components/widgets/CachePerformance.svelte";
    import CacheStats from "$lib/components/widgets/CacheStats.svelte";
    import DataTransfer from "$lib/components/widgets/DataTransfer.svelte";
    import RequestCoalescing from "$lib/components/widgets/RequestCoalescing.svelte";
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
            { Comp: CacheEfficiency, size: { width: 2, height: 2 } },
            { Comp: CachePerformance, size: { width: 2, height: 2 } },
            { Comp: TotalRequests, size: { width: 2, height: 2 } },
            { Comp: RequestCoalescing, size: { width: 4, height: 3 } },
            { Comp: DataTransfer, size: { width: 2, height: 2 } },
            { Comp: SystemInfo, size: { width: 1, height: 1 } },
            { Comp: CacheStats, size: { width: 2, height: 3 } },
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
