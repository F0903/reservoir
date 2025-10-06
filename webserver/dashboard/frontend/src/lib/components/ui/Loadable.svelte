<script lang="ts">
    import type { Snippet } from "svelte";
    import ErrorBox from "./ErrorBox.svelte";
    import type { Loadable } from "$lib/utils/loadable";

    let { state, loadable, children }: { state?: any; loadable: Loadable; children: Snippet } =
        $props();

    const loadState = $derived.by(() => loadable.getLoadableState());
</script>

{#if loadState.tag === "loading"}
    <div class="loading-box"></div>
{:else if loadState.tag === "error"}
    <ErrorBox>{loadState.errorMsg}</ErrorBox>
{:else if state !== null}
    {@render children()}
{/if}

<style>
    @keyframes loading {
        0% {
            backdrop-filter: var(--bg-brightness-from);
        }
        50% {
            backdrop-filter: var(--bg-brightness-to);
        }
        100% {
            backdrop-filter: var(--bg-brightness-from);
        }
    }

    .loading-box {
        --bg-brightness-to: var(--loading-bg-brightness-to, brightness(0.8));
        --bg-brightness-from: var(--loading-bg-brightness-from, brightness(0.6));

        height: 100px;
        border-radius: 10px;
        padding: 1rem;
        text-align: center;

        animation-name: loading;
        animation-duration: 1.5s;
        animation-timing-function: ease-in-out;
        animation-iteration-count: infinite;
    }
</style>
