<script lang="ts">
    import Card from "$lib/components/ui/Card.svelte";

    export type CapacityMetricFooterItem = {
        label: string;
        value: string | number;
    };

    const {
        label,
        value,
        percent,
        progressLabel = label,
        footerItems = [],
    }: {
        label: string;
        value: string | number;
        percent: number;
        progressLabel?: string;
        footerItems?: CapacityMetricFooterItem[];
    } = $props();

    const clampedPercent = $derived(
        Number.isFinite(percent) ? Math.min(100, Math.max(0, percent)) : 0,
    );
</script>

<div class="capacity-metric-card-wrapper">
    <Card
        --card-height="100%"
        --card-width="100%"
        --card-padding="var(--capacity-metric-padding, 0.8rem)"
        --card-background="var(--capacity-metric-background, var(--primary-600))"
        --card-border-radius="var(--capacity-metric-border-radius, 8px)"
        --card-justify-content="space-around"
        --card-gap="0.7rem"
    >
        <div class="capacity-topline">
            <div class="value-block">
                <span class="capacity-label">{label}</span>
                <strong>{value}</strong>
            </div>
            <span class="fill-percent">{clampedPercent.toFixed(1)}%</span>
        </div>

        <div class="capacity-track" aria-label={progressLabel}>
            <div class="capacity-fill" style:width={`${clampedPercent}%`}></div>
        </div>

        {#if footerItems.length > 0}
            <div class="capacity-meta">
                {#each footerItems as item (item.label)}
                    <span>{item.label} <strong>{item.value}</strong></span>
                {/each}
            </div>
        {/if}
    </Card>
</div>

<style>
    .capacity-metric-card-wrapper {
        height: 100%;
        min-height: 0;
        width: 100%;
    }

    .capacity-topline {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 1rem;
    }

    .value-block {
        display: flex;
        flex-direction: column;
        gap: 0.4rem;
        min-width: 0;
    }

    .capacity-label {
        color: rgba(255, 255, 255, 0.4);
        font-size: 0.58rem;
        font-weight: 700;
        letter-spacing: 0.08em;
        line-height: 1;
        text-transform: uppercase;
    }

    .value-block strong {
        color: var(--capacity-metric-value-color, var(--secondary-300));
        font-size: 1.2rem;
        font-weight: 700;
        line-height: 1;
        white-space: nowrap;
    }

    .fill-percent {
        color: var(--capacity-metric-value-color, var(--secondary-300));
        font-family: "Chivo Mono Variable", monospace;
        font-size: 1rem;
        font-weight: 700;
    }

    .capacity-track {
        height: 0.5rem;
        overflow: hidden;
        border-radius: 999px;
        background-color: var(--primary-700);
        border: 1px solid rgba(255, 255, 255, 0.08);
    }

    .capacity-fill {
        height: 100%;
        border-radius: inherit;
        background-color: var(--capacity-metric-value-color, var(--secondary-300));
        transition: width 160ms ease;
    }

    .capacity-meta {
        display: flex;
        flex-wrap: wrap;
        gap: 0.45rem 0.8rem;
        color: rgba(255, 255, 255, 0.4);
        font-size: 0.68rem;
        font-weight: 600;
    }

    .capacity-meta strong {
        color: var(--capacity-metric-value-color, var(--secondary-300));
        font-weight: 700;
    }

    @media (max-width: 768px) {
        .capacity-metric-card-wrapper {
            --capacity-metric-padding: 0.7rem;
        }

        .value-block strong {
            font-size: 1.15rem;
        }
    }
</style>
