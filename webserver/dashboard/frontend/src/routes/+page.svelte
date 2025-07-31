<script>
    import FlexGrid from "$lib/components/layout/FlexGrid.svelte";
    import TotalRequests from "$lib/components/widgets/TotalRequests.svelte";
    import SystemInfo from "$lib/components/widgets/SystemInfo.svelte";
    import CacheEfficiency from "$lib/components/widgets/CacheEfficiency.svelte";
    import CachePerformance from "$lib/components/widgets/CachePerformance.svelte";
    import CacheStats from "$lib/components/widgets/CacheStats.svelte";
    import DataTransfer from "$lib/components/widgets/DataTransfer.svelte";

    let { data } = $props();
</script>

<main class="dashboard">
    <FlexGrid>
        <CacheEfficiency metrics={data.metrics} />
        <CachePerformance metrics={data.metrics} />
        <TotalRequests metrics={data.metrics} />
        <CacheStats metrics={data.metrics} />
        <DataTransfer metrics={data.metrics} />
        <SystemInfo metrics={data.metrics} />
    </FlexGrid>
    <div class="widgets"></div>
</main>

<style>
    .dashboard {
        padding: 1.5rem;
    }

    @media (max-width: 768px) {
        .dashboard {
            padding: 0;
        }

        .widgets {
            grid-template-columns: 1fr;
            gap: 1rem;
        }
    }
</style>
