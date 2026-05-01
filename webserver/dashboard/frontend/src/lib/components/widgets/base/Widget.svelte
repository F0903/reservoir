<script lang="ts">
    import Card from "$lib/components/ui/Card.svelte";
    import type { Snippet } from "svelte";

    let {
        title,
        headerControls,
        children,
    }: {
        title: string;
        headerControls?: Snippet;
        children: Snippet;
    } = $props();
</script>

<Card
    --card-width="var(--widget-width, 100%)"
    --card-height="var(--widget-height, 100%)"
    --card-padding="0"
    --card-background="var(--primary-500)"
>
    <div class="widget-header">
        <h2 class="title">{title}</h2>
        {#if headerControls}
            <div class="header-controls">
                {@render headerControls()}
            </div>
        {/if}
    </div>
    <div class="widget-content">
        {@render children()}
    </div>
</Card>

<style>
    .widget-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.75rem;
        padding: 0.75rem 1rem;
        background-color: rgba(255, 255, 255, 0.03);
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
        border-radius: 15px 15px 0 0;
    }

    .widget-content {
        display: flex;
        flex-direction: column;
        flex-grow: 1;
        min-height: 0;
        padding: 1rem;
        overflow: hidden;
    }

    .title {
        font-size: 0.85rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: var(--secondary-300);
        text-align: left;
        min-width: 0;
    }

    .header-controls {
        display: flex;
        align-items: center;
        flex-shrink: 0;
    }

    @media (max-width: 768px) {
        .widget-content {
            padding: 0.5rem;
        }

        .widget-header {
            padding: 0.4rem 0.6rem;
        }

        .title {
            font-size: 0.75rem;
        }
    }
</style>
