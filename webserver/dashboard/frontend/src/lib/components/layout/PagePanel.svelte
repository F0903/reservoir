<script lang="ts">
    import type { Snippet } from "svelte";

    let {
        gap = "1rem",
        padding = "1rem",
        mobilePadding = padding,
        children,
    }: {
        gap?: string;
        padding?: string;
        mobilePadding?: string;
        children: Snippet;
    } = $props();
</script>

<section
    class="page-panel"
    style={`--page-panel-gap: ${gap}; --page-panel-padding: ${padding}; --page-panel-mobile-padding: ${mobilePadding};`}
>
    {@render children()}
</section>

<style>
    .page-panel {
        display: flex;
        flex-direction: column;
        gap: var(--page-panel-gap);
        width: 100%;
        padding: var(--page-panel-padding);
        border: 1px solid rgba(255, 255, 255, 0.06);
        border-radius: 8px;
        background-color: var(--primary-500);
    }

    @media (max-width: 768px) {
        .page-panel {
            padding: var(--page-panel-mobile-padding);
        }
    }
</style>
