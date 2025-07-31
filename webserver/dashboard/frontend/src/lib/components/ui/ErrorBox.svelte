<script lang="ts">
    let { children } = $props();
</script>

<div class="error">
    <h2>Error!</h2>
    {@render children()}
</div>

<style>
    .error {
        background-color: var(--error-background-color);
        color: var(--error-text-color);
        border: 1px solid var(--error-border-color);
        padding: 0.75rem;
        border-radius: 8px;
    }

    h2 {
        margin-bottom: 0.5rem;
    }
</style>
