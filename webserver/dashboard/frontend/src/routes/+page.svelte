<script lang="ts">
    import TotalRequests from "$lib/components/widgets/TotalRequests.svelte";
    import SystemInfo from "$lib/components/widgets/SystemInfo.svelte";
    import CacheEfficiency from "$lib/components/widgets/CacheEfficiency.svelte";
    import CachePerformance from "$lib/components/widgets/CachePerformance.svelte";
    import CacheStats from "$lib/components/widgets/CacheStats.svelte";
    import DataTransfer from "$lib/components/widgets/DataTransfer.svelte";
    import ComponentMasonryGrid from "$lib/components/layout/ComponentMasonryGrid.svelte";
    import { getContext, onMount } from "svelte";
    import type { MetricsProvider } from "$lib/providers/metrics.svelte";

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
    <ComponentMasonryGrid
        components={[
            CacheEfficiency,
            CachePerformance,
            TotalRequests,
            CacheStats,
            DataTransfer,
            SystemInfo,
        ]}
    />
</main>

<style>
    .dashboard {
        padding: 1.5rem;
    }

    @media (max-width: 768px) {
        .dashboard {
            padding: 0;
        }
    }
</style>
