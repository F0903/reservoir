<script lang="ts">
    import Tooltip from "$lib/components/ui/Tooltip.svelte";
    import type {
        DashboardEditableGridElement,
        DashboardResizeMode,
    } from "$lib/dashboard/dashboard-editor";

    let {
        element,
        onDragStart,
        onResizeStart,
    }: {
        element: DashboardEditableGridElement;
        onDragStart: (_event: PointerEvent, _element: DashboardEditableGridElement) => void;
        onResizeStart: (
            _event: PointerEvent,
            _element: DashboardEditableGridElement,
            _mode: DashboardResizeMode,
        ) => void;
    } = $props();

    const label = $derived(element.label ?? "widget");
</script>

<span class="drag-control">
    <Tooltip text={`Drag ${label}`} align="end">
        <button
            class="drag-handle"
            onpointerdown={(event) => onDragStart(event, element)}
            aria-label={`Drag ${label}`}
        >
            <span class="drag-dots" aria-hidden="true"></span>
        </button>
    </Tooltip>
</span>
<span class="resize-control resize-control-right">
    <Tooltip text={`Resize ${label} width`} align="end">
        <button
            class="resize-handle resize-right"
            onpointerdown={(event) => onResizeStart(event, element, "width")}
            aria-label={`Resize ${label} width`}
        ></button>
    </Tooltip>
</span>
<span class="resize-control resize-control-bottom">
    <Tooltip text={`Resize ${label} height`} align="end">
        <button
            class="resize-handle resize-bottom"
            onpointerdown={(event) => onResizeStart(event, element, "height")}
            aria-label={`Resize ${label} height`}
        ></button>
    </Tooltip>
</span>
<span class="resize-control resize-control-corner">
    <Tooltip text={`Resize ${label}`} align="end">
        <button
            class="resize-handle resize-corner"
            onpointerdown={(event) => onResizeStart(event, element, "both")}
            aria-label={`Resize ${label}`}
        ></button>
    </Tooltip>
</span>
<div class="size-badge" aria-hidden="true">
    {element.span.width}x{element.span.height}
</div>

<style>
    .drag-control,
    .resize-control {
        position: absolute;
        z-index: 10;
        display: block;
    }

    .drag-control {
        top: 0.46rem;
        left: 50%;
        transform: translateX(-50%);
    }

    .resize-control :global(.tooltip) {
        display: block;
        width: 100%;
        height: 100%;
        line-height: 0;
    }

    .drag-handle {
        display: grid;
        place-items: center;
        width: 2.2rem;
        height: 1.2rem;
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 999px;
        background-color: color-mix(in srgb, var(--primary-700) 84%, transparent);
        box-shadow: 0 5px 14px rgba(0, 0, 0, 0.28);
        color: rgba(255, 255, 255, 0.55);
        cursor: grab;
        transition:
            background-color 120ms ease,
            color 120ms ease,
            border-color 120ms ease;
    }

    .drag-handle:hover {
        border-color: rgba(255, 255, 255, 0.14);
        background-color: color-mix(in srgb, var(--primary-700) 72%, var(--secondary-800));
        color: var(--secondary-300);
    }

    .drag-handle:active {
        cursor: grabbing;
    }

    .drag-dots,
    .drag-dots::before,
    .drag-dots::after {
        display: block;
        width: 0.22rem;
        height: 0.22rem;
        border-radius: 999px;
        background-color: currentColor;
    }

    .drag-dots {
        position: relative;
    }

    .drag-dots::before,
    .drag-dots::after {
        content: "";
        position: absolute;
        top: 0;
    }

    .drag-dots::before {
        left: -0.45rem;
    }

    .drag-dots::after {
        right: -0.45rem;
    }

    .resize-control-right {
        top: 2.5rem;
        right: -0.35rem;
        bottom: 2.5rem;
        width: 0.7rem;
    }

    .resize-control-bottom {
        right: 2.5rem;
        bottom: -0.35rem;
        left: 2.5rem;
        height: 0.7rem;
    }

    .resize-control-corner {
        right: -0.625rem;
        bottom: -0.625rem;
        width: 1.25rem;
        height: 1.25rem;
    }

    .resize-handle {
        position: relative;
        display: block;
        width: 100%;
        height: 100%;
        border: none;
        border-radius: 999px;
        background: transparent;
        padding: 0;
        color: rgba(255, 255, 255, 0.46);
        transition:
            color 120ms ease,
            opacity 120ms ease;
    }

    .resize-right {
        cursor: ew-resize;
    }

    .resize-right::before {
        content: "";
        position: absolute;
        top: 0.35rem;
        bottom: 0.35rem;
        left: 0.35rem;
        width: 0.18rem;
        min-height: 1.8rem;
        border-radius: 999px;
        background-color: currentColor;
        transform: translateX(-50%);
    }

    .resize-bottom {
        cursor: ns-resize;
    }

    .resize-bottom::before {
        content: "";
        position: absolute;
        right: 0.35rem;
        left: 0.35rem;
        top: 0.35rem;
        height: 0.18rem;
        min-width: 1.8rem;
        border-radius: 999px;
        background-color: currentColor;
        transform: translateY(-50%);
    }

    .resize-right:hover,
    .resize-bottom:hover {
        color: var(--secondary-300);
    }

    .resize-corner {
        border-radius: 0;
        cursor: nwse-resize;
        color: rgba(255, 255, 255, 0.52);
    }

    .resize-corner::before {
        content: "";
        position: absolute;
        right: 50%;
        bottom: 50%;
        width: 0.72rem;
        height: 0.72rem;
        border-right: 0.18rem solid currentColor;
        border-bottom: 0.18rem solid currentColor;
        border-bottom-right-radius: 0.58rem;
    }

    .resize-corner:hover {
        color: var(--secondary-300);
    }

    .size-badge {
        position: absolute;
        right: 0.68rem;
        bottom: 0.68rem;
        z-index: 9;
        padding: 0.18rem 0.38rem;
        border: 1px solid rgba(255, 255, 255, 0.06);
        border-radius: 6px;
        background-color: color-mix(in srgb, var(--primary-700) 82%, transparent);
        color: rgba(255, 255, 255, 0.56);
        font-family: "Chivo Mono Variable", monospace;
        font-size: 0.6rem;
        font-weight: 700;
        line-height: 1;
    }

    @media (max-width: 768px) {
        .drag-control {
            top: 0.32rem;
        }

        .drag-handle {
            width: 2rem;
            height: 1.1rem;
        }

        .size-badge {
            right: 0.55rem;
            bottom: 0.55rem;
        }
    }
</style>
