<script lang="ts">
    import type { DashboardDragGhostState } from "$lib/dashboard/dashboard-editor";

    let {
        ghost,
    }: {
        ghost: DashboardDragGhostState;
    } = $props();
</script>

<div
    class="drag-ghost"
    style:width={`${ghost.width}px`}
    style:height={`${ghost.height}px`}
    style:transform={`translate3d(${ghost.pointerX - ghost.offsetX}px, ${ghost.pointerY - ghost.offsetY}px, 0)`}
    aria-hidden="true"
>
    <div class="drag-ghost-header">
        <span>{ghost.label}</span>
        <strong>{ghost.spanLabel}</strong>
    </div>
    <div class="drag-ghost-body">
        <span class="drag-ghost-line"></span>
        <span class="drag-ghost-line short"></span>
    </div>
</div>

<style>
    .drag-ghost {
        position: fixed;
        top: 0;
        left: 0;
        z-index: 1000;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        pointer-events: none;
        border: 1px solid color-mix(in srgb, var(--secondary-300) 52%, transparent);
        border-radius: 15px;
        background-color: color-mix(in srgb, var(--primary-500) 74%, transparent);
        box-shadow:
            0 18px 48px rgba(0, 0, 0, 0.38),
            inset 0 0 0 1px rgba(255, 255, 255, 0.04);
        opacity: 0.9;
        backdrop-filter: blur(8px);
    }

    .drag-ghost-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.75rem;
        padding: 0.7rem 0.9rem;
        border-bottom: 1px solid rgba(255, 255, 255, 0.06);
        background-color: rgba(255, 255, 255, 0.035);
        color: var(--secondary-300);
        font-size: 0.76rem;
        font-weight: 800;
        letter-spacing: 0.04em;
        line-height: 1;
        text-transform: uppercase;
    }

    .drag-ghost-header span {
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .drag-ghost-header strong {
        flex-shrink: 0;
        color: rgba(255, 255, 255, 0.55);
        font-family: "Chivo Mono Variable", monospace;
        font-size: 0.62rem;
        font-weight: 700;
    }

    .drag-ghost-body {
        display: flex;
        flex: 1;
        flex-direction: column;
        justify-content: flex-end;
        min-height: 0;
        padding: 0.9rem;
        background:
            linear-gradient(
                135deg,
                color-mix(in srgb, var(--secondary-300) 12%, transparent),
                transparent 46%
            ),
            repeating-linear-gradient(
                -45deg,
                rgba(255, 255, 255, 0.025) 0,
                rgba(255, 255, 255, 0.025) 1px,
                transparent 1px,
                transparent 9px
            );
    }

    .drag-ghost-line {
        display: block;
        width: 48%;
        height: 0.36rem;
        margin-top: 0.35rem;
        border-radius: 999px;
        background-color: rgba(255, 255, 255, 0.11);
    }

    .drag-ghost-line.short {
        width: 32%;
        background-color: rgba(255, 255, 255, 0.075);
    }
</style>
