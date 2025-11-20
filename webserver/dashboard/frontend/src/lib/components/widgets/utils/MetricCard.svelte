<script lang="ts">
    import Card from "$lib/components/ui/Card.svelte";

    type DynamicFontSize = {
        base: number;
        min: number;
        max: number;
    };

    const defaultValueSize: DynamicFontSize = {
        base: 1.5,
        min: 0.8,
        max: 2.4,
    };

    const defaultLabelSize: DynamicFontSize = {
        base: 0.9,
        min: 0.55,
        max: 1.1,
    };

    const {
        label,
        value,
        dynamicValueSize = defaultValueSize,
        dynamicLabelSize = defaultLabelSize,
    } = $props();

    const clampSize = (size: number, min: number, max: number) =>
        Math.max(min, Math.min(max, size));

    const computeFontSize = (
        content: string | number,
        base: number,
        step: number,
        min: number,
        max: number,
    ) => {
        const normalized = String(content ?? "").replace(/\s+/g, "");
        return clampSize(base - normalized.length * step, min, max);
    };

    const valueFontSize = $derived(
        computeFontSize(
            value,
            dynamicValueSize.base,
            0.08,
            dynamicValueSize.min,
            dynamicValueSize.max,
        ),
    );
    const labelFontSize = $derived(
        computeFontSize(
            label,
            dynamicLabelSize.base,
            0.012,
            dynamicLabelSize.min,
            dynamicLabelSize.max,
        ),
    );
</script>

<div
    class="metric-card-wrapper"
    style={`--auto-value-size:${valueFontSize}rem; --auto-label-size:${labelFontSize}rem;`}
>
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
        --auto-value-size: clamp(0.9rem, 14cqw, 1.1rem);
        --auto-label-size: clamp(0.6rem, 8cqw, 0.8rem);
    }

    .metric-value {
        font-size: var(--metric-value-size, var(--auto-value-size));
        font-weight: var(--metric-value-weight, bold);
        color: var(--metric-value-color, var(--secondary-400));
        margin-bottom: 0.25rem;
        line-height: 1.15;
        word-break: break-word;
    }

    .metric-label {
        font-size: var(--metric-label-size, var(--auto-label-size));
        color: var(--metric-label-color, var(--primary-200));
        text-transform: uppercase;
        letter-spacing: 0.05em;
        line-height: 1.1;
        word-break: break-word;
    }
</style>
