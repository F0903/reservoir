<!-- A utility component for polling a function at a specified interval as long as its children are visible -->

<script lang="ts">
    import { visible } from "$lib/attachments/visible";
    import { log } from "$lib/utils/logger";

    let { isVisible = $bindable(true), pollFn, pollInterval = 5000, children } = $props();

    let intervalId: number | null = null;

    function visibilityChanged(state: boolean) {
        isVisible = state;
    }

    function startPolling() {
        if (intervalId !== null) return;

        pollFn(); // Initial fetch before starting the interval
        intervalId = setInterval(pollFn, pollInterval);
        log.debug(`Started polling with interval ${pollInterval} ms. Id=${intervalId}`);
    }

    function stopPolling() {
        if (intervalId === null) {
            return;
        }

        log.debug(`Stopping polling. Id=${intervalId}`);

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

<div class="poller" {@attach visible(visibilityChanged)}>
    {@render children()}
</div>
