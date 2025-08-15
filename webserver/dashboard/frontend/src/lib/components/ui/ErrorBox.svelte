<script lang="ts">
    let { children } = $props();
</script>

<div class="error">
    <h2 class="error-title">Error!</h2>
    {@render children()}
</div>

<style>
    .error {
        background-color: var(--error-background-color);
        color: var(--error-text-color);
        border: 1px solid var(--error-border-color);
        padding: 1rem;
        border-radius: 8px;
        font-size: 0.9rem;
    }

    .error-title {
        margin-bottom: 0.5rem;
        font-size: 1.15rem;
        font-weight: 700;
    }
</style>
