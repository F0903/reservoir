<script lang="ts" generics="Comp extends Component">
    import type { Component } from "svelte";

    let {
        components,
    }: {
        components: Comp[];
    } = $props();
</script>

<div class="grid">
    {#each components as Component (Component)}
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
