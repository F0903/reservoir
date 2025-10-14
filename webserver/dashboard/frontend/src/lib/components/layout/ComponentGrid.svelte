<script lang="ts">
    import type { Component } from "svelte";

    type MasonryComponent = Component<any, Record<string, any>, any>;

    let {
        components,
    }: {
        components: MasonryComponent[];
    } = $props();
</script>

<div class="grid">
    {#each components as Component}
        <div class="grid-item">
            <Component />
        </div>
    {/each}
</div>

<style>
    .grid {
        display: flex;
        flex-direction: row;
        flex-wrap: wrap;
        gap: var(--mason-grid-gap, 15px);
        justify-content: center; /* Centers content when it wraps */

        margin: auto;
    }
</style>
