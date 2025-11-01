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
    --card-padding="var(--widget-padding, 1.5rem)"
>
    <h2 class="title">{title}</h2>
    <div class="widget-content">
        {@render children()}
    </div>
</Card>

<style>
    .widget-content {
        flex-grow: 1;
        min-height: 0;
    }

    .title {
        margin-top: -8px;
        margin-bottom: 0.5rem;

        font-size: var(--widget-title-size, 1.2rem);
        font-weight: 600;
        text-align: center;
    }
</style>
