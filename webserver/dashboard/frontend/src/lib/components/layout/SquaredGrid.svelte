<script lang="ts">
    import { onMount, type Component } from "svelte";
    import type { IntRange } from "$lib/utils/type-utils";

    type GridElemSize = IntRange<1, 5>;

    let {
        elements,
        cellSize = 150,
        gap = 15,
    }: {
        elements: {
            Comp: Component;
            size: { width: GridElemSize; height: GridElemSize };
        }[];
        cellSize?: number;
        gap?: number;
    } = $props();

    let grid: HTMLDivElement;

    let parentWidth: number | undefined = $state();

    let initialized = $state(false);

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

        initialized = true;

        return () => {
            parentObserver.disconnect();
        };
    });

    $effect(() => {
        if (!initialized) return;

        const columns = Math.floor(parentWidth! / (cellSize + gap));

        grid.style.gridTemplateColumns = `repeat(${columns}, ${cellSize}px)`;
        grid.style.gridAutoRows = `${cellSize}px`;
        grid.style.gap = `${gap}px`;
    });
</script>

<div class="grid" bind:this={grid}>
    {#if initialized}
        {#each elements as { Comp, size } (Comp)}
            <div
                class="grid-elem"
                style="grid-column: span {size.width}; grid-row: span {size.height};"
            >
                <Comp />
            </div>
        {/each}
    {/if}
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
