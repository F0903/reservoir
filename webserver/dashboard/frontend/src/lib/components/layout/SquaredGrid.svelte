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

    const initialColumns = 4;

    let grid: HTMLDivElement;

    let parentWidth: number | undefined = $state();
    let columns = $state(initialColumns);
    let measuredCellSize = $state(1);
    let hasMeasured = $state(false);
    const effectiveCellSize = $derived(hasMeasured ? measuredCellSize : cellSize);
    const placedElements = $derived(resolveGridElements(elements, columns));

    onMount(() => {
        if (!grid.parentElement) return;

        setParentWidth(grid.parentElement.offsetWidth);

        const parentObserver = new ResizeObserver((entries) => {
            const entry = entries[0];
            setParentWidth(entry.contentRect.width);
        });
        parentObserver.observe(grid.parentElement);

        return () => {
            parentObserver.disconnect();
        };
    });

    $effect(() => {
        if (!parentWidth) return;

        const dimensions = gridDimensions(parentWidth);
        measuredCellSize = dimensions.cellSize;
        columns = dimensions.columns;
    });

    function setParentWidth(width: number) {
        if (!Number.isFinite(width) || width <= 0) return;

        parentWidth = width;
        hasMeasured = true;
    }

    function gridDimensions(width: number) {
        // On mobile, we use a smaller cell size to fit more content or at least not overflow.
        const nextCellSize = viewport.isMobile
            ? Math.max(1, Math.min(cellSize, (width - gap) / 2))
            : cellSize;

        return {
            cellSize: nextCellSize,
            // Subtract one gap from total width because gaps only exist BETWEEN columns.
            columns: Math.max(1, Math.floor((width - gap) / (nextCellSize + gap)) + 1),
        };
    }

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
    aria-busy={!hasMeasured}
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
            {#if hasMeasured && children}
                {@render children(element)}
            {:else if hasMeasured}
                <Comp />
            {:else}
                <div class="grid-placeholder" aria-hidden="true">
                    <span class="placeholder-title"></span>
                    <span class="placeholder-line"></span>
                    <span class="placeholder-line short"></span>
                </div>
            {/if}
        </div>
    {/each}
</div>

<style>
    .grid-elem {
        width: 100%;
        height: 100%;
    }

    .grid-placeholder {
        position: relative;
        display: flex;
        flex-direction: column;
        justify-content: flex-end;
        gap: 0.55rem;
        width: 100%;
        height: 100%;
        overflow: hidden;
        padding: 1rem;
        border: 1px solid var(--primary-500);
        border-radius: 15px;
        background-color: var(--primary-500);
    }

    .placeholder-title,
    .placeholder-line {
        display: block;
        border-radius: 999px;
        background-color: rgba(255, 255, 255, 0.08);
    }

    .placeholder-title {
        position: absolute;
        top: 0.95rem;
        left: 1rem;
        width: 38%;
        height: 0.58rem;
        background-color: color-mix(in srgb, var(--secondary-300) 22%, transparent);
    }

    .placeholder-line {
        width: 52%;
        height: 0.44rem;
    }

    .placeholder-line.short {
        width: 34%;
        background-color: rgba(255, 255, 255, 0.055);
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
