<script>
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import Button from "../ui/input/Button.svelte";

    let { url, children = undefined } = $props();

    const isCurrent = $derived(page.url.pathname === url);
    const backgroundColor = $derived(isCurrent ? "var(--tertiary-400)" : "var(--secondary-400)");

    async function onClick() {
        await goto(url);
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
