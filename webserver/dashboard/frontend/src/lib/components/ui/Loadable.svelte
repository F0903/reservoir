<script lang="ts" generics="T">
    import type { Snippet } from "svelte";
    import ErrorBox from "./ErrorBox.svelte";

    let {
        state,
        error,
        children,
    }: {
        state: T | null | undefined;
        error?: string | undefined | null;
        children: Snippet<[data: NonNullable<T>]>;
    } = $props();
</script>

{#if error}
    <ErrorBox>{error}</ErrorBox>
{:else if state === null || state === undefined}
    <div class="loading-box"></div>
{:else}
    {@render children(state as NonNullable<T>)}
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
