<script lang="ts">
    import { onMount, type Component } from "svelte";
    import type { IntRange } from "$lib/utils/type-utils";
    import { viewport } from "$lib/utils/viewport.svelte";

    type CellSpan = IntRange<1, 5>;

    let {
        elements,
        cellSize = 150,
        gap = 15,
    }: {
        elements: {
            Comp: Component;
            span: { width: CellSpan; height: CellSpan };
            mobileSpan?: { width?: CellSpan; height?: CellSpan };
        }[];
        cellSize?: number;
        gap?: number;
    } = $props();

    let grid: HTMLDivElement;

    let parentWidth: number | undefined = $state();

    onMount(() => {
        if (!grid.parentElement) return;

        // Set initial parent width
        parentWidth = grid.parentElement.offsetWidth;

        const parentObserver = new ResizeObserver((entries) => {
            // Update parent width when resized
            const entry = entries[0];
            parentWidth = entry.contentRect.width;
        });
        parentObserver.observe(grid.parentElement);

        return () => {
            parentObserver.disconnect();
        };
    });

    $effect(() => {
        if (!parentWidth) return;

        // On mobile, we use a smaller cell size to fit more content or at least not overflow
        const effectiveCellSize = viewport.isMobile
            ? Math.min(cellSize, (parentWidth - gap) / 2)
            : cellSize;
        // Subtract one gap from total width because gaps only exist BETWEEN columns
        const columns = Math.max(
            1,
            Math.floor((parentWidth - gap) / (effectiveCellSize + gap)) + 1,
        );

        // Explicitly set the amount of columns, and use gridAutoRows which can then automatically create rows as needed.
        grid.style.gridTemplateColumns = `repeat(${columns}, 1fr)`;
        grid.style.gridAutoRows = `${effectiveCellSize}px`;
        grid.style.gap = `${gap}px`;

        // Update each grid element to cap its span to the number of columns
        const elements = grid.querySelectorAll<HTMLDivElement>(".grid-elem");
        elements.forEach((el) => {
            const spanWidth = parseInt(
                (viewport.isMobile ? el.dataset.mobileSpanWidth : null) ||
                    el.dataset.spanWidth ||
                    "1",
            );
            const spanHeight = parseInt(
                (viewport.isMobile ? el.dataset.mobileSpanHeight : null) ||
                    el.dataset.spanHeight ||
                    "1",
            );
            el.style.gridColumn = `span ${Math.min(spanWidth, columns)}`;
            el.style.gridRow = `span ${spanHeight}`;
        });
    });
</script>

<div class="grid" bind:this={grid}>
    {#each elements as { Comp, span: size, mobileSpan: mSize } (Comp)}
        <div
            class="grid-elem"
            data-span-width={size.width}
            data-span-height={size.height}
            data-mobile-span-width={mSize?.width}
            data-mobile-span-height={mSize?.height}
        >
            <Comp />
        </div>
    {/each}
</div>

<style>
    .grid-elem {
        width: 100%;
        height: 100%;
    }

    .grid {
        display: grid;
        grid-auto-flow: row dense;

        width: 100%;
        height: fit-content;

        margin-left: auto;
        margin-right: auto;
    }
</style>
