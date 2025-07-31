<script>
    import FlexGrid from "$lib/components/layout/FlexGrid.svelte";
    import CacheMetrics from "$lib/components/widgets/CacheMetrics.svelte";
    import RequestMetrics from "$lib/components/widgets/RequestMetrics.svelte";
    import SystemMetrics from "$lib/components/widgets/SystemMetrics.svelte";
</script>

<main class="dashboard">
    <FlexGrid>
        <SystemMetrics />
        <RequestMetrics />
        <CacheMetrics />
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
