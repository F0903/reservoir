<script lang="ts">
    let { children } = $props();
</script>

<div class="card">
    {@render children()}
</div>

<style>
    .card {
        width: var(--card-width, inherit);
        height: var(--card-height, inherit);

        text-align: var(--card-text-align, inherit);

        padding: var(--card-padding, 1.5rem);
        border: var(--card-border, none);
        border-radius: var(--card-border-radius, 15px);
        background-color: var(--card-background, var(--primary-400));

        display: flex;
        flex-direction: column;
        justify-content: var(--card-justify-content, space-around);
    }
</style>
