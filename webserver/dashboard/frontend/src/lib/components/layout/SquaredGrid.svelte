<script lang="ts">
    import { onMount, type Component } from "svelte";
    import type { IntRange } from "$lib/utils/type-utils";

    type CellSpan = IntRange<1, 5>;

    let {
        elements,
        cellSize = 150,
        gap = 15,
    }: {
        elements: {
            Comp: Component;
            span: { width: CellSpan; height: CellSpan };
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
        const columns = Math.floor(parentWidth! / (cellSize + gap));

        // Explicitly set the amount of columns, and use gridAutoRows which can then automatically create rows as needed.
        grid.style.gridTemplateColumns = `repeat(${columns}, ${cellSize}px)`;
        grid.style.gridAutoRows = `${cellSize}px`;
        grid.style.gap = `${gap}px`;
    });
</script>

<div class="grid" bind:this={grid}>
    {#each elements as { Comp, span: size } (Comp)}
        <div
            class="grid-elem"
            style="grid-column: span {size.width}; grid-row: span {size.height};"
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

        width: fit-content;
        height: fit-content;

        margin-left: auto;
        margin-right: auto;
    }
</style>
