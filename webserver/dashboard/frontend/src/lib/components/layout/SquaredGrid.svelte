<script lang="ts">
    import { onMount, type Component, type Snippet } from "svelte";
    import type { IntRange } from "$lib/utils/type-utils";
    import { viewport } from "$lib/utils/viewport.svelte";

    type CellSpan = IntRange<1, 5>;

    type SquaredGridElement = {
        id?: string;
        label?: string;
        Comp: Component;
        span: { width: CellSpan; height: CellSpan };
        mobileSpan?: { width?: CellSpan; height?: CellSpan };
        position?: { column: number; row: number };
    };

    type PlacedSquaredGridElement = SquaredGridElement & {
        gridPosition: { column: number; row: number };
        gridSpan: { width: number; height: number };
    };

    let {
        elements,
        cellSize = 150,
        gap = 15,
        children,
    }: {
        elements: SquaredGridElement[];
        cellSize?: number;
        gap?: number;
        children?: Snippet<[SquaredGridElement]>;
    } = $props();

    let grid: HTMLDivElement;

    let parentWidth: number | undefined = $state();
    let columns = $state(1);
    let effectiveCellSize = $state(1);
    const placedElements = $derived(resolveGridElements(elements, columns));

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

        // On mobile, we use a smaller cell size to fit more content or at least not overflow.
        effectiveCellSize = viewport.isMobile
            ? Math.max(1, Math.min(cellSize, (parentWidth - gap) / 2))
            : cellSize;
        // Subtract one gap from total width because gaps only exist BETWEEN columns
        columns = Math.max(1, Math.floor((parentWidth - gap) / (effectiveCellSize + gap)) + 1);
    });

    function spanWidth(element: SquaredGridElement) {
        return viewport.isMobile
            ? (element.mobileSpan?.width ?? element.span.width)
            : element.span.width;
    }

    function spanHeight(element: SquaredGridElement) {
        return viewport.isMobile
            ? (element.mobileSpan?.height ?? element.span.height)
            : element.span.height;
    }

    function occupiedCellKey(column: number, row: number) {
        return `${column}:${row}`;
    }

    function canPlaceElement(
        occupiedCells: Set<string>,
        position: { column: number; row: number },
        span: { width: number; height: number },
        gridColumns: number,
    ) {
        if (position.column < 1 || position.column + span.width - 1 > gridColumns) return false;

        for (let row = position.row; row < position.row + span.height; row += 1) {
            for (let column = position.column; column < position.column + span.width; column += 1) {
                if (occupiedCells.has(occupiedCellKey(column, row))) return false;
            }
        }

        return true;
    }

    function occupyElementCells(
        occupiedCells: Set<string>,
        position: { column: number; row: number },
        span: { width: number; height: number },
    ) {
        for (let row = position.row; row < position.row + span.height; row += 1) {
            for (let column = position.column; column < position.column + span.width; column += 1) {
                occupiedCells.add(occupiedCellKey(column, row));
            }
        }
    }

    function clampPosition(
        position: { column: number; row: number },
        span: { width: number; height: number },
        gridColumns: number,
    ) {
        const maxColumn = Math.max(1, gridColumns - span.width + 1);

        return {
            column: Math.min(maxColumn, Math.max(1, Math.round(position.column))),
            row: Math.max(1, Math.round(position.row)),
        };
    }

    function firstAvailablePosition(
        occupiedCells: Set<string>,
        span: { width: number; height: number },
        gridColumns: number,
    ) {
        const maxColumn = Math.max(1, gridColumns - span.width + 1);

        for (let row = 1; ; row += 1) {
            for (let column = 1; column <= maxColumn; column += 1) {
                const position = { column, row };
                if (canPlaceElement(occupiedCells, position, span, gridColumns)) return position;
            }
        }
    }

    function resolveGridElements(
        gridElements: SquaredGridElement[],
        gridColumns: number,
    ): PlacedSquaredGridElement[] {
        if (gridColumns < 1) return [];

        const occupiedCells = new Set<string>();
        return gridElements.map((element) => {
            const gridSpan = {
                width: Math.min(spanWidth(element), gridColumns),
                height: spanHeight(element),
            };
            const desiredPosition = element.position
                ? clampPosition(element.position, gridSpan, gridColumns)
                : undefined;
            const gridPosition =
                desiredPosition &&
                canPlaceElement(occupiedCells, desiredPosition, gridSpan, gridColumns)
                    ? desiredPosition
                    : firstAvailablePosition(occupiedCells, gridSpan, gridColumns);

            occupyElementCells(occupiedCells, gridPosition, gridSpan);

            return {
                ...element,
                gridPosition,
                gridSpan,
            };
        });
    }
</script>

<div
    class="grid"
    bind:this={grid}
    data-squared-grid
    data-grid-columns={columns}
    data-grid-cell-size={effectiveCellSize}
    data-grid-gap={gap}
    style:grid-template-columns={`repeat(${columns}, 1fr)`}
    style:grid-auto-rows={`${effectiveCellSize}px`}
    style:gap={`${gap}px`}
>
    {#each placedElements as element (element.id ?? element.Comp)}
        {@const { Comp } = element}
        <div
            class="grid-elem"
            style:grid-column={`${element.gridPosition.column} / span ${element.gridSpan.width}`}
            style:grid-row={`${element.gridPosition.row} / span ${element.gridSpan.height}`}
        >
            {#if children}
                {@render children(element)}
            {:else}
                <Comp />
            {/if}
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
