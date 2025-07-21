<script lang="ts">
    let { children } = $props();
</script>

<div class="side-nav">
    {@render children()}
</div>

<style>
    .side-nav {
        background-color: var(--quaternary-color);
        padding: 1rem;

        height: 100%;
        width: 200px;

        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: start;
        gap: 1rem;
    }
</style>
