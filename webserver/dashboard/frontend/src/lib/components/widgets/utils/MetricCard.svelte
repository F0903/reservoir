<script lang="ts">
    import Card from "$lib/components/ui/Card.svelte";
    import type { Component } from "svelte";

    const {
        label,
        value,
        icon: Icon = null,
    }: {
        label: string;
        value: string | number;
        icon?: Component | null;
    } = $props();
</script>

<div class="metric-card-wrapper">
    <Card
        --card-height="100%"
        --card-width="var(--metric-width, 100%)"
        --card-text-align="var(--metric-text-align, left)"
        --card-padding="var(--metric-padding, 0.75rem 1rem)"
        --card-background="var(--metric-background, var(--primary-600))"
        --card-border="var(--metric-border, 1px solid var(--primary-500))"
        --card-border-radius="var(--metric-border-radius, 12px)"
        --card-justify-content="flex-start"
        --card-gap="0.25rem"
    >
        <div class="accent-bar"></div>
        <div class="metric-header">
            <span class="metric-label">{label}</span>
            {#if Icon}
                <div class="icon-container hide-on-mobile">
                    <Icon size={14} />
                </div>
            {/if}
        </div>
        <div class="metric-value">{value}</div>
    </Card>
</div>

<style>
    .metric-card-wrapper {
        container-type: inline-size;
        display: flex;
        height: 100%;
        width: 100%;
        transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
        position: relative;

        flex: 1;
    }

    .metric-card-wrapper:hover {
        transform: translateY(-1px);
        filter: brightness(1.05);
    }

    .accent-bar {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 2px;
        background-color: var(--metric-value-color, var(--secondary-300));
        opacity: 0.6;
        border-radius: 12px 12px 0 0;
    }

    .metric-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        width: 100%;
        margin-bottom: 0.1rem;
    }

    .icon-container {
        color: var(--metric-value-color, var(--secondary-300));
        opacity: 0.5;
    }

    .metric-value {
        /* Balanced scaling using cqmin to handle both width and height constraints */
        font-size: var(--metric-value-size, clamp(0.5rem, 13cqmin, 1.4rem));
        font-weight: 700;
        color: var(--metric-value-color, var(--secondary-300));
        line-height: 1.1;

        white-space: var(--metric-value-whitespace, nowrap);
    }

    .metric-label {
        font-size: var(--metric-label-size, clamp(0.5rem, 10cqmin, 0.7rem));
        color: var(--metric-label-color, rgba(255, 255, 255, 0.4));
        text-transform: uppercase;
        letter-spacing: 0.08em;
        font-weight: 600;
        line-height: 1;
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
    }

    @media (max-width: 768px) {
        .metric-card-wrapper {
            --metric-padding: 0.4rem 0.6rem;
        }
    }
</style>
