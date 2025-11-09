<script lang="ts">
    import Card from "$lib/components/ui/Card.svelte";

    let { label, value } = $props();
</script>

<div class="metric-card-wrapper">
    <Card
        --card-height="var(--metric-height, 100%)"
        --card-width="var(--metric-width, 100%)"
        --card-text-align="var(--metric-text-align, center)"
        --card-padding="var(--metric-padding, 1rem)"
        --card-background="var(--metric-background, var(--primary-600))"
        --card-border="var(--metric-border, 1px solid var(--primary-500))"
        --card-border-radius="var(--metric-border-radius, 8px)"
    >
        <div class="metric-value">{value}</div>
        <div class="metric-label">{label}</div>
    </Card>
</div>

<style>
    .metric-card-wrapper {
        container-type: inline-size;
        height: 100%;
        width: 100%;
    }

    .metric-value {
        font-size: var(--metric-value-size, clamp(0.9rem, 14cqw, 1.1rem));
        font-weight: var(--metric-value-weight, bold);
        color: var(--metric-value-color, var(--secondary-400));
        margin-bottom: 0.25rem;
    }

    .metric-label {
        font-size: var(--metric-label-size, clamp(0.6rem, 8cqw, 0.8rem));
        color: var(--metric-label-color, var(--primary-200));
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }
</style>
