<script lang="ts">
    import TotalRequests from "$lib/components/widgets/HTTPRequests.svelte";
    import SystemInfo from "$lib/components/widgets/SystemInfo.svelte";
    import CacheEfficiency from "$lib/components/widgets/CacheEfficiency.svelte";
    import CachePerformance from "$lib/components/widgets/CachePerformance.svelte";
    import CacheStats from "$lib/components/widgets/CacheStats.svelte";
    import DataTransfer from "$lib/components/widgets/DataTransfer.svelte";
    import RequestCoalescing from "$lib/components/widgets/RequestCoalescing.svelte";
    import ComponentGrid from "$lib/components/layout/ComponentGrid.svelte";
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
    <ComponentGrid
        components={[
            CacheEfficiency,
            CachePerformance,
            TotalRequests,
            RequestCoalescing,
            DataTransfer,
            SystemInfo,
            CacheStats,
        ]}
    />
</main>

<style>
    .dashboard {
        padding: 1.5rem;
    }

    @media (max-width: var(--mobile-cutoff)) {
        .dashboard {
            padding: 0;
        }
    }
</style>
