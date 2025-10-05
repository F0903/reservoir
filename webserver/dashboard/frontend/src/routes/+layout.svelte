<script lang="ts">
    import "../global.css";
    import "@fontsource-variable/open-sans";
    import "@fontsource-variable/chivo-mono";
    import { setContext } from "svelte";
    import { SettingsProvider } from "$lib/providers/settings/settings-provider.svelte";
    import { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { ToastProvider } from "$lib/providers/toast-provider.svelte";
    import { log } from "$lib/utils/logger";

    let { children } = $props();

    const settings = setContext("settings", new SettingsProvider());
    const metrics = setContext("metrics", new MetricsProvider(settings, fetch));
    const toast = setContext("toast", new ToastProvider());

    function onAsyncError(event: PromiseRejectionEvent) {
        event.preventDefault();

        log.error("Unhandled promise rejection caught: ", event.reason, event.promise);

        toast.show({
            type: "error",
            message: event.reason ?? "An unexpected error occurred.",
            durationMs: 10000,
            dismissText: "Dismiss",
        });
    }

    function onError(
        eventOrMessage: Event | string,
        source?: string,
        lineno?: number,
        colno?: number,
        error?: Error,
    ) {
        let message = "";
        if (eventOrMessage instanceof Event) {
            log.debug("Unhandled error parameter was event.");
        } else {
            message = eventOrMessage;
        }

        message ??= error?.message ?? "An unexpected error occurred.";
        log.error("Unhandled error caught: ", message, source, lineno, colno, error);

        toast.show({
            type: "error",
            message,
            durationMs: 10000,
            dismissText: "Dismiss",
        });
    }
</script>

<svelte:window onunhandledrejection={onAsyncError} onerror={onError} />

{@render children()}
