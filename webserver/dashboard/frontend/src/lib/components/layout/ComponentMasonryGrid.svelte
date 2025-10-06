<script lang="ts">
    import { mount, type Component } from "svelte";
    import { log } from "$lib/utils/logger";

    let { components, columns = 4 }: { components: Component<any, any, any>[]; columns?: number } =
        $props();

    let gridBase: HTMLElement | null = $state(null);

    let columnElems: HTMLElement[] | null = null;

    function contructColumns() {
        if (!gridBase || columnElems !== null) return;

        log.debug("Constructing columns for WidgetGrid");

        columnElems = [];
        for (let i = 0; i < columns; i++) {
            const column = document.createElement("div");
            column.classList.add("grid-column");
            gridBase.appendChild(column);
        }
    }

    function mountComponents() {
        gridBase = gridBase!; // Gridbase is guaranteed to be non-null here

        for (let i = 0; i < components.length; i++) {
            const comp = components[i];

            const columnIndex = i % columns;
            const column = gridBase.children[columnIndex] as HTMLElement;

            mount(comp, {
                target: column,
            });
        }
    }

    $effect(() => {
        if (!gridBase) return;

        if (!columnElems) {
            contructColumns();
        }

        mountComponents();
    });
</script>

<div class="grid" bind:this={gridBase}></div>

<style>
    .grid {
        /* Create a flex row to contain the columns */
        display: flex;
        flex-direction: row;
        justify-content: center;
        gap: var(--widget-grid-gap, 1rem);
        flex-wrap: wrap;

        width: fit-content;
        margin: auto;
    }

    :global(.grid .grid-column) {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: var(--widget-grid-gap, 1rem);
    }
</style>
