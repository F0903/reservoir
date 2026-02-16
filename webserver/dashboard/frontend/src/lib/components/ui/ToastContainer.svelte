<script lang="ts">
    import { flip } from "svelte/animate";
    import Toast from "./Toast.svelte";
    import { getToastProvider } from "$lib/context";

    const toastProvider = getToastProvider();
</script>

<div class="toast-container">
    {#each toastProvider.toasts as entry (entry.id)}
        <div animate:flip={{ duration: 200 }}>
            <Toast {...entry.props} handle={entry.handle} />
        </div>
    {/each}
</div>

<style>
    .toast-container {
        position: fixed;
        bottom: 40px;
        left: 50%;
        transform: translateX(-50%);

        display: flex;
        flex-direction: column-reverse;
        gap: 15px;

        z-index: 9999;
        pointer-events: none; /* Allow clicking through the container, but Toast will override this with auto */
    }
</style>
