<script lang="ts">
    import DashboardDragGhost from "$lib/components/dashboard/DashboardDragGhost.svelte";
    import DashboardLayoutToolbar from "$lib/components/dashboard/DashboardLayoutToolbar.svelte";
    import DashboardWidgetEditorControls from "$lib/components/dashboard/DashboardWidgetEditorControls.svelte";
    import SquaredGrid from "$lib/components/layout/SquaredGrid.svelte";
    import type {
        DashboardDragGhostState,
        DashboardEditableGridElement,
        DashboardGridMetrics,
        DashboardResizeMode,
    } from "$lib/dashboard/dashboard-editor";
    import { dashboardGridPositionFromPoint } from "$lib/dashboard/dashboard-editor";
    import {
        defaultDashboardLayout,
        packDashboardLayout,
        placeDashboardWidget,
        resolveDashboardWidgetId,
        setDashboardWidgetSpan,
        type DashboardWidgetId,
        type DashboardWidgetLayout,
    } from "$lib/dashboard/dashboard-layout";
    import type { DashboardGridElement } from "$lib/dashboard/dashboard-widgets";
    import { onDestroy } from "svelte";

    type ResizeState = {
        id: DashboardWidgetId;
        mode: DashboardResizeMode;
        startX: number;
        startY: number;
        startWidth: number;
        startHeight: number;
        lastWidth: number;
        lastHeight: number;
        unitWidth: number;
        unitHeight: number;
    };

    type DragPointer = {
        clientX: number;
        clientY: number;
    };

    let {
        elements,
        layout,
        onLayoutChange,
        onRefresh,
        refreshing = false,
        gap = 15,
    }: {
        elements: DashboardGridElement[];
        layout: DashboardWidgetLayout[];
        onLayoutChange: (_layout: DashboardWidgetLayout[]) => void;
        onRefresh?: () => void | Promise<void>;
        refreshing?: boolean;
        gap?: number;
    } = $props();

    let editing = $state(false);
    let draggingWidgetId = $state<DashboardWidgetId | null>(null);
    let dragGhost = $state<DashboardDragGhostState | null>(null);
    let resizingWidgetId = $state<DashboardWidgetId | null>(null);
    let resizeState: ResizeState | null = null;
    let pendingDragPointer: DragPointer | null = null;
    let dragFrame: number | null = null;

    onDestroy(() => {
        stopWidgetDrag();
        stopResize();
    });

    function saveLayout(nextLayout: DashboardWidgetLayout[]) {
        onLayoutChange(nextLayout);
    }

    function currentGridMetrics(): DashboardGridMetrics | null {
        if (typeof document === "undefined") return null;

        const grid = document.querySelector<HTMLElement>("[data-squared-grid]");
        if (!grid) return null;

        const columns = Number(grid.dataset.gridColumns);
        const rowHeight = Number(grid.dataset.gridCellSize);
        const gridGap = Number(grid.dataset.gridGap);
        if (!Number.isFinite(columns) || columns < 1) return null;
        if (!Number.isFinite(rowHeight) || rowHeight < 1) return null;
        if (!Number.isFinite(gridGap) || gridGap < 0) return null;

        const rect = grid.getBoundingClientRect();
        const columnWidth = (rect.width - gridGap * (columns - 1)) / columns;
        if (!Number.isFinite(columnWidth) || columnWidth < 1) return null;

        return {
            left: rect.left,
            top: rect.top,
            columnWidth,
            rowHeight,
            gap: gridGap,
            columns,
        };
    }

    function dragGhostTopLeftPoint(pointer: DragPointer) {
        if (!dragGhost) {
            return { x: pointer.clientX, y: pointer.clientY };
        }

        return {
            x: pointer.clientX - dragGhost.offsetX,
            y: pointer.clientY - dragGhost.offsetY,
        };
    }

    function startWidgetDrag(event: PointerEvent, element: DashboardEditableGridElement) {
        const widgetId = resolveDashboardWidgetId(element.id);
        if (!editing || !widgetId || event.button !== 0) return;
        if (typeof window === "undefined") return;

        const shell = (event.currentTarget as HTMLElement).closest<HTMLElement>(".widget-shell");
        const widgetLayout = layout.find((item) => item.id === widgetId);
        if (!shell || !widgetLayout) return;

        const rect = shell.getBoundingClientRect();
        event.preventDefault();
        event.stopPropagation();
        draggingWidgetId = widgetId;
        dragGhost = {
            label: element.label ?? widgetId,
            spanLabel: `${widgetLayout.span.width}x${widgetLayout.span.height}`,
            width: rect.width,
            height: rect.height,
            pointerX: event.clientX,
            pointerY: event.clientY,
            offsetX: event.clientX - rect.left,
            offsetY: event.clientY - rect.top,
        };
        window.addEventListener("pointermove", dragWidget);
        window.addEventListener("pointerup", stopWidgetDrag, { once: true });
    }

    function updateDragGhost(pointer: DragPointer) {
        if (!dragGhost) return;

        dragGhost = {
            ...dragGhost,
            pointerX: pointer.clientX,
            pointerY: pointer.clientY,
        };
    }

    function dragWidget(event: PointerEvent) {
        if (!draggingWidgetId) return;
        if (typeof window === "undefined") return;

        pendingDragPointer = {
            clientX: event.clientX,
            clientY: event.clientY,
        };
        if (dragFrame !== null) return;

        dragFrame = window.requestAnimationFrame(flushWidgetDrag);
    }

    function flushWidgetDrag() {
        dragFrame = null;
        const pointer = pendingDragPointer;
        pendingDragPointer = null;
        if (!draggingWidgetId || !pointer) return;

        updateDragGhost(pointer);
        const widgetLayout = layout.find((item) => item.id === draggingWidgetId);
        const metrics = currentGridMetrics();
        if (!widgetLayout || !metrics) return;

        const point = dragGhostTopLeftPoint(pointer);
        const position = dashboardGridPositionFromPoint(metrics, point, widgetLayout.span);
        if (
            widgetLayout.position?.column === position.column &&
            widgetLayout.position?.row === position.row
        ) {
            return;
        }

        saveLayout(placeDashboardWidget(layout, draggingWidgetId, position, metrics.columns));
    }

    function stopWidgetDrag() {
        if (typeof window !== "undefined") {
            window.removeEventListener("pointermove", dragWidget);
            window.removeEventListener("pointerup", stopWidgetDrag);
            if (dragFrame !== null) {
                window.cancelAnimationFrame(dragFrame);
            }
        }
        dragFrame = null;
        pendingDragPointer = null;
        draggingWidgetId = null;
        dragGhost = null;
    }

    function startResize(
        event: PointerEvent,
        element: DashboardEditableGridElement,
        mode: DashboardResizeMode,
    ) {
        const widgetId = resolveDashboardWidgetId(element.id);
        if (!editing || !widgetId || event.button !== 0) return;
        if (typeof window === "undefined") return;

        const shell = (event.currentTarget as HTMLElement).closest<HTMLElement>(".widget-shell");
        const widgetLayout = layout.find((item) => item.id === widgetId);
        if (!shell || !widgetLayout) return;

        const rect = shell.getBoundingClientRect();
        const unitWidth = (rect.width + gap) / widgetLayout.span.width;
        const unitHeight = (rect.height + gap) / widgetLayout.span.height;

        event.preventDefault();
        event.stopPropagation();
        resizeState = {
            id: widgetId,
            mode,
            startX: event.clientX,
            startY: event.clientY,
            startWidth: widgetLayout.span.width,
            startHeight: widgetLayout.span.height,
            lastWidth: widgetLayout.span.width,
            lastHeight: widgetLayout.span.height,
            unitWidth,
            unitHeight,
        };
        resizingWidgetId = widgetId;
        window.addEventListener("pointermove", resizeWidget);
        window.addEventListener("pointerup", stopResize, { once: true });
    }

    function resizeWidget(event: PointerEvent) {
        if (!resizeState) return;

        const widthDelta =
            resizeState.mode === "width" || resizeState.mode === "both"
                ? Math.round((event.clientX - resizeState.startX) / resizeState.unitWidth)
                : 0;
        const heightDelta =
            resizeState.mode === "height" || resizeState.mode === "both"
                ? Math.round((event.clientY - resizeState.startY) / resizeState.unitHeight)
                : 0;

        const nextWidth = resizeState.startWidth + widthDelta;
        const nextHeight = resizeState.startHeight + heightDelta;
        if (nextWidth === resizeState.lastWidth && nextHeight === resizeState.lastHeight) {
            return;
        }

        const resizedLayout = setDashboardWidgetSpan(layout, resizeState.id, {
            width: nextWidth,
            height: nextHeight,
        });
        const metrics = currentGridMetrics();
        saveLayout(metrics ? packDashboardLayout(resizedLayout, metrics.columns) : resizedLayout);
        resizeState.lastWidth = nextWidth;
        resizeState.lastHeight = nextHeight;
    }

    function stopResize() {
        if (typeof window !== "undefined") {
            window.removeEventListener("pointermove", resizeWidget);
            window.removeEventListener("pointerup", stopResize);
        }
        resizingWidgetId = null;
        resizeState = null;
    }

    function resetLayout() {
        saveLayout(defaultDashboardLayout());
    }
</script>

<div class="layout-editor">
    <DashboardLayoutToolbar
        {editing}
        {refreshing}
        onEdit={() => (editing = true)}
        {onRefresh}
        onReset={resetLayout}
        onSave={() => (editing = false)}
    />

    <SquaredGrid {elements} {gap}>
        {#snippet children(element)}
            {@const Comp = element.Comp}
            <div
                class="widget-shell"
                class:editing
                class:dragging={draggingWidgetId === element.id}
                class:resizing={resizingWidgetId === element.id}
                data-dashboard-widget-id={element.id}
            >
                <Comp />
                {#if editing}
                    <DashboardWidgetEditorControls
                        {element}
                        onDragStart={startWidgetDrag}
                        onResizeStart={startResize}
                    />
                {/if}
            </div>
        {/snippet}
    </SquaredGrid>
</div>

{#if dragGhost}
    <DashboardDragGhost ghost={dragGhost} />
{/if}

<style>
    .layout-editor {
        overflow-anchor: none;
    }

    .widget-shell {
        position: relative;
        width: 100%;
        height: 100%;
        transition:
            opacity 120ms ease,
            transform 120ms ease;
    }

    .widget-shell.editing::after {
        content: "";
        position: absolute;
        inset: 0;
        pointer-events: none;
        border: 1px solid color-mix(in srgb, var(--secondary-300) 42%, transparent);
        border-radius: 15px;
        box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.04);
    }

    .widget-shell.resizing::after {
        border-color: color-mix(in srgb, var(--tertiary-400) 58%, transparent);
        box-shadow:
            inset 0 0 0 1px rgba(255, 255, 255, 0.06),
            0 0 0 1px color-mix(in srgb, var(--tertiary-400) 30%, transparent);
    }

    .widget-shell.dragging {
        opacity: 0.38;
        transform: scale(0.99);
    }
</style>
