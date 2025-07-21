<script>
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import Button from "../ui/Button.svelte";

    let { url, children = undefined } = $props();

    const isCurrent = $derived(page.url.pathname === url);
    const backgroundColor = $derived(
        isCurrent ? "var(--sidenav-selected)" : "var(--sidenav-normal)",
    );

    async function onClick() {
        await goto(url);
    }
</script>

<Button {onClick} --btn-background-color={backgroundColor}>
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
