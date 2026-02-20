<script lang="ts">
    import CacheEfficiency from "$lib/components/widgets/CacheEfficiency.svelte";
    import CacheLatency from "$lib/components/widgets/CacheLatency.svelte";
    import CacheStats from "$lib/components/widgets/CacheStats.svelte";
    import DataTransfer from "$lib/components/widgets/DataTransfer.svelte";
    import RequestVolume from "$lib/components/widgets/RequestVolume.svelte";
    import RequestCoalescing from "$lib/components/widgets/RequestCoalescing.svelte";
    import RequestLatency from "$lib/components/widgets/RequestLatency.svelte";
    import ResponseStatus from "$lib/components/widgets/ResponseStatus.svelte";
    import SystemInfo from "$lib/components/widgets/SystemInfo.svelte";
    import SquaredGrid from "$lib/components/layout/SquaredGrid.svelte";
    import { getMetricsProvider } from "$lib/context";
    import { onMount } from "svelte";

    const metrics = getMetricsProvider();

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
            { Comp: CacheEfficiency, span: { width: 3, height: 3 }, mobileSpan: { width: 1 } },
            { Comp: CacheLatency, span: { width: 2, height: 2 } },
            {
                Comp: RequestLatency,
                span: { width: 3, height: 3 },
                mobileSpan: { width: 3, height: 2 },
            },
            { Comp: RequestVolume, span: { width: 3, height: 3 }, mobileSpan: { width: 1 } },
            { Comp: ResponseStatus, span: { width: 2, height: 2 } },
            {
                Comp: RequestCoalescing,
                span: { width: 4, height: 3 },
                mobileSpan: { height: 2 },
            },
            { Comp: DataTransfer, span: { width: 2, height: 3 }, mobileSpan: { height: 3 } },
            { Comp: SystemInfo, span: { width: 1, height: 3 }, mobileSpan: { width: 1 } },
            {
                Comp: CacheStats,
                span: { width: 3, height: 2 },
                mobileSpan: { width: 2, height: 3 },
            },
        ]}
    />
</main>

<style>
    main {
        height: fit-content;
        width: 100%;
    }

    .dashboard {
        padding: 1.5rem;
    }

    @media (max-width: 768px) {
        .dashboard {
            margin: 0;
            padding: 0;
        }
    }
</style>
