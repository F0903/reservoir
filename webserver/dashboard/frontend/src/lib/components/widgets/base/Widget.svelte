<script lang="ts">
    import Card from "$lib/components/ui/Card.svelte";
    import type { Snippet } from "svelte";

    let {
        title,
        children,
    }: {
        title: string;
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
    </div>
    <div class="widget-content">
        {@render children()}
    </div>
</Card>

<style>
    .widget-header {
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
