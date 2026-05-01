<script lang="ts">
    import DashboardLayoutEditor from "$lib/components/dashboard/DashboardLayoutEditor.svelte";
    import { createDashboardGridElements } from "$lib/dashboard/dashboard-widgets";
    import type { DashboardWidgetLayout } from "$lib/dashboard/dashboard-layout";
    import { getMetricsProvider, getSettingsProvider } from "$lib/context";
    import { onMount } from "svelte";

    const metrics = getMetricsProvider();
    const settings = getSettingsProvider();

    const gridElements = $derived(
        createDashboardGridElements(settings.dashboardSettings.fields.layout),
    );

    onMount(() => {
        metrics.refreshMetrics();
        metrics.startRefresh();

        return () => {
            metrics.stopRefresh();
        };
    });

    function saveLayout(layout: DashboardWidgetLayout[]) {
        settings.dashboardSettings.fields.layout = layout;
    }
</script>

<main class="dashboard">
    <DashboardLayoutEditor
        elements={gridElements}
        layout={settings.dashboardSettings.fields.layout}
        onLayoutChange={saveLayout}
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
