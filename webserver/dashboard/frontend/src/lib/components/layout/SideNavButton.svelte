<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import type { Snippet } from "svelte";
    import Button from "../ui/input/Button.svelte";

    let {
        url,
        children = undefined,
        onClick: onCustomClick = undefined,
    }: { url: string; children?: Snippet; onClick?: () => void } = $props();

    const isCurrent = $derived(page.url.pathname === url);
    const backgroundColor = $derived(isCurrent ? "var(--tertiary-400)" : "var(--secondary-400)");

    async function onClick() {
        if (onCustomClick) {
            onCustomClick();
        }
        let to = resolve("/");
        to += url.startsWith("/") ? url.substring(1) : url;
        await goto(to);
    }
</script>

<Button {onClick} disabled={isCurrent} --btn-background-color={backgroundColor} --btn-width="100%">
    <div class="content">
        {@render children?.()}
    </div>
</Button>

<style>
    .content {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 0.3rem;
    }
</style>
