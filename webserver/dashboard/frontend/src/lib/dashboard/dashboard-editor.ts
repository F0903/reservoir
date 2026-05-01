import type { DashboardGridPosition } from "$lib/dashboard/dashboard-layout";

export type DashboardResizeMode = "width" | "height" | "both";

export type DashboardDragGhostState = {
    label: string;
    spanLabel: string;
    width: number;
    height: number;
    pointerX: number;
    pointerY: number;
    offsetX: number;
    offsetY: number;
};

export type DashboardEditableGridElement = {
    id?: string;
    label?: string;
    span: {
        width: number;
        height: number;
    };
};

export type DashboardGridMetrics = {
    left: number;
    top: number;
    columnWidth: number;
    rowHeight: number;
    gap: number;
    columns: number;
};

export type DashboardGridPoint = {
    x: number;
    y: number;
};

export type DashboardGridSpan = {
    width: number;
    height: number;
};

export function dashboardGridPositionFromPoint(
    metrics: DashboardGridMetrics,
    point: DashboardGridPoint,
    span: DashboardGridSpan,
): DashboardGridPosition {
    const columnStride = metrics.columnWidth + metrics.gap;
    const rowStride = metrics.rowHeight + metrics.gap;
    const effectiveWidth = Math.min(span.width, metrics.columns);
    const maxColumn = Math.max(1, metrics.columns - effectiveWidth + 1);

    return {
        column: Math.min(
            maxColumn,
            Math.max(1, Math.round((point.x - metrics.left) / columnStride) + 1),
        ),
        row: Math.max(1, Math.round((point.y - metrics.top) / rowStride) + 1),
    };
}
