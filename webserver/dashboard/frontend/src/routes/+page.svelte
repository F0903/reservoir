<script>
    import FlexGrid from "$lib/components/layout/FlexGrid.svelte";
    import TotalRequests from "$lib/components/widgets/TotalRequests.svelte";
    import SystemInfo from "$lib/components/widgets/SystemInfo.svelte";

    let { data } = $props();
</script>

<main class="dashboard">
    <FlexGrid>
        <SystemInfo metrics={data.metrics} />
        <TotalRequests metrics={data.metrics} />
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
