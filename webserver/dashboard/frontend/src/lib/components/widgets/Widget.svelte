<script lang="ts">
    import { visible } from "$lib/attachments/visible";

    let { title, isVisible = $bindable(true), onPoll, children } = $props();

    const POLL_INTERVAL_MS = 10000; // This should be configurable at some point

    let intervalId: number | null = null;

    function visibilityChanged(state: boolean) {
        isVisible = state;
    }

    function startPolling() {
        if (intervalId) return;

        onPoll();

        intervalId = setInterval(onPoll, POLL_INTERVAL_MS);
    }

    function stopPolling() {
        if (!intervalId) {
            return;
        }

        clearInterval(intervalId);
        intervalId = null;
    }

    $effect(() => {
        // Stop polling if the widget is no longer visible
        if (isVisible) {
            startPolling();
        } else {
            stopPolling();
        }
    });
</script>

<div class="widget" {@attach visible(visibilityChanged)}>
    <h2 class="title">{title}</h2>
    <div class="widget-content">
        {@render children()}
    </div>
</div>

<style>
    .title {
        font-size: 1.5rem;
        font-weight: 600;
        margin-bottom: 0.5rem;
    }

    .widget {
        width: fit-content;
        height: fit-content;

        padding: 1.5rem;
        border-radius: 15px;
        background-color: var(--primary-400);
    }
</style>
