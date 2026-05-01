import CacheEfficiency from "$lib/components/widgets/CacheEfficiency.svelte";
import CacheLatency from "$lib/components/widgets/CacheLatency.svelte";
import CacheStorage from "$lib/components/widgets/CacheStorage.svelte";
import CacheStats from "$lib/components/widgets/CacheStats.svelte";
import DataTransfer from "$lib/components/widgets/DataTransfer.svelte";
import RequestCoalescing from "$lib/components/widgets/RequestCoalescing.svelte";
import RequestLatency from "$lib/components/widgets/RequestLatency.svelte";
import RequestVolume from "$lib/components/widgets/RequestVolume.svelte";
import ResponseStatus from "$lib/components/widgets/ResponseStatus.svelte";
import SystemInfo from "$lib/components/widgets/SystemInfo.svelte";
import type { Component } from "svelte";
import {
    dashboardWidgetDefinitions,
    type DashboardWidgetId,
    type DashboardWidgetLayout,
} from "./dashboard-layout";

export type DashboardGridElement = DashboardWidgetLayout & {
    label: string;
    Comp: Component;
};

const widgetComponents: Record<DashboardWidgetId, Component> = {
    "cache-efficiency": CacheEfficiency,
    "cache-latency": CacheLatency,
    "request-latency": RequestLatency,
    "request-volume": RequestVolume,
    "response-status": ResponseStatus,
    "request-coalescing": RequestCoalescing,
    "data-transfer": DataTransfer,
    "system-info": SystemInfo,
    "cache-stats": CacheStats,
    "cache-storage": CacheStorage,
};

const widgetLabels = new Map(
    dashboardWidgetDefinitions.map((definition) => [definition.id, definition.label]),
);

export function getDashboardWidgetLabel(id: DashboardWidgetId): string {
    return widgetLabels.get(id) ?? id;
}

export function createDashboardGridElements(
    layout: DashboardWidgetLayout[],
): DashboardGridElement[] {
    return layout.map((item) => ({
        ...item,
        label: getDashboardWidgetLabel(item.id),
        Comp: widgetComponents[item.id],
    }));
}
