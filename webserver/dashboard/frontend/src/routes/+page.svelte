<script>
    import CacheMetrics from "$lib/components/widgets/CacheMetrics.svelte";
    import RequestMetrics from "$lib/components/widgets/RequestMetrics.svelte";
    import SystemMetrics from "$lib/components/widgets/SystemMetrics.svelte";
</script>

<main class="dashboard">
    <div class="widgets">
        <SystemMetrics />
        <RequestMetrics />
        <CacheMetrics />
    </div>
</main>

<style>
    .dashboard {
        padding: 1.5rem;
    }

    .widgets {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
        gap: 1.5rem;
        align-items: start;
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
