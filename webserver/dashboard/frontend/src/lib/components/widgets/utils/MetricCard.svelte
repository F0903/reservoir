<script lang="ts">
    import Card from "$lib/components/ui/Card.svelte";

    let { label, value } = $props();
</script>

<Card
    --card-height="var(--metric-height, auto)"
    --card-width="var(--metric-width, auto)"
    --card-text-align="var(--metric-text-align, center)"
    --card-padding="var(--metric-padding, 1rem)"
    --card-background="var(--metric-background, var(--primary-600))"
    --card-border="var(--metric-border, 1px solid var(--primary-500))"
    --card-border-radius="var(--metric-border-radius, 8px)"
>
    <div class="metric-value">{value}</div>
    <div class="metric-label">{label}</div>
</Card>

<style>
    .metric-value {
        font-size: var(--metric-value-size, 1.5rem);
        font-weight: var(--metric-value-weight, bold);
        color: var(--metric-value-color, var(--secondary-400));
        margin-bottom: 0.25rem;
    }

    .metric-label {
        font-size: var(--metric-label-size, 0.875rem);
        color: var(--metric-label-color, var(--primary-200));
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }
</style>
