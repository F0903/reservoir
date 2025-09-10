<script lang="ts">
    let { children } = $props();
</script>

<h1 class="page-title">
    {@render children()}
</h1>

<style>
    .page-title {
        font-size: 1.75rem;
        font-weight: bold;
        color: var(--tertiary-400);
        margin-bottom: var(--pagetitle-margin-bottom, 1rem);
    }
</style>
