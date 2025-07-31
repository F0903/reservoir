<script lang="ts">
    let { children } = $props();
</script>

<div class="grid">
    {@render children()}
</div>

<style>
    .grid {
        display: flex;
        flex-wrap: wrap;
        gap: 1rem;
        justify-content: center; /* Centers content when it wraps */

        margin: auto;
    }
</style>
