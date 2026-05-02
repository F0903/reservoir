<script lang="ts">
    import type { Snippet } from "svelte";

    let {
        title,
        description,
        icon,
        meta,
    }: {
        title: string;
        description?: string;
        icon?: Snippet;
        meta?: Snippet;
    } = $props();
</script>

<div class="panel-heading">
    {@render icon?.()}
    <div class="title-block">
        <h2>{title}</h2>
        {#if description}
            <p>{description}</p>
        {/if}
    </div>
    {@render meta?.()}
</div>

<style>
    .panel-heading {
        display: flex;
        align-items: flex-start;
        gap: 0.65rem;
        color: var(--secondary-300);
    }

    .title-block {
        min-width: 0;
    }

    h2 {
        color: var(--secondary-300);
        font-size: 0.92rem;
        font-weight: 800;
        line-height: 1.1;
        text-transform: uppercase;
    }

    p {
        margin-top: 0.35rem;
        color: rgba(255, 255, 255, 0.52);
        font-size: 0.85rem;
        line-height: 1.4;
    }
</style>
