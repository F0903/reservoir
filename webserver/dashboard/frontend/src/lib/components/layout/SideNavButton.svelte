<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import type { Snippet } from "svelte";
    import Button from "../ui/input/Button.svelte";

    let { url, children = undefined }: { url: string; children?: Snippet } = $props();

    const isCurrent = $derived(page.url.pathname === url);
    const backgroundColor = $derived(isCurrent ? "var(--tertiary-400)" : "var(--secondary-400)");

    async function onClick() {
        let to = resolve("/");
        to += url.startsWith("/") ? url : `/${url}`;
        await goto(to);
    }
</script>

<Button {onClick} disabled={isCurrent} --btn-background-color={backgroundColor}>
    <div class="content">
        {@render children?.()}
    </div>
</Button>

<style>
    .content {
        display: flex;
        align-items: center;
        gap: 0.3rem;
    }
</style>
