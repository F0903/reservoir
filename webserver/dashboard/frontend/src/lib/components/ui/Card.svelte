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
        border: 1px solid rgba(255, 255, 255, 0.05);
        border-radius: var(--card-border-radius, 15px);
        background-color: var(--card-background, var(--primary-400));

        display: flex;
        flex-direction: column;
        justify-content: var(--card-justify-content, space-around);
        gap: var(--card-gap, 5px);

        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
    }
</style>
