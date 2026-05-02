<script lang="ts">
    import type { Snippet } from "svelte";

    let {
        maxWidth = "1100px",
        gap = "1rem",
        children,
    }: {
        maxWidth?: string;
        gap?: string;
        children: Snippet;
    } = $props();
</script>

<div
    class="content-shell"
    style={`--content-shell-max-width: ${maxWidth}; --content-shell-gap: ${gap};`}
>
    {@render children()}
</div>

<style>
    .content-shell {
        display: flex;
        flex-direction: column;
        gap: var(--content-shell-gap);
        width: min(100%, var(--content-shell-max-width));
        margin-inline: auto;
    }
</style>
