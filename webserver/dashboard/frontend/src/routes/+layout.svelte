<script lang="ts">
    import "../global.css";
    import "@fontsource-variable/open-sans";
    import "@fontsource-variable/chivo-mono";
    import { SettingsProvider } from "$lib/providers/settings/settings-provider.svelte";
    import { MetricsProvider } from "$lib/providers/metrics/metrics-provider.svelte";
    import { ToastProvider } from "$lib/providers/toast/toast-provider.svelte";
    import { log } from "$lib/utils/logger";
    import {
        setSettingsProvider,
        setMetricsProvider,
        setToastProvider,
        setAuthProvider,
    } from "$lib/context";
    import ToastContainer from "$lib/components/ui/ToastContainer.svelte";
    import { AuthProvider } from "$lib/providers/auth/auth-provider.svelte";

    let { children } = $props();

    // Initialize all providers so child components can access them
    const settings = setSettingsProvider(new SettingsProvider());
    const _metrics = setMetricsProvider(new MetricsProvider(settings, fetch));
    const toasts = setToastProvider(new ToastProvider());
    const _session = setAuthProvider(new AuthProvider());

    function onAsyncError(event: PromiseRejectionEvent) {
        event.preventDefault();

        log.error("Unhandled promise rejection caught: ", event.reason, event.promise);
        toasts.error(event.reason ?? "An unexpected error occurred.");
    }

    function onError(
        eventOrMessage: Event | string,
        source?: string,
        lineno?: number,
        colno?: number,
    ) {
        let message = null;
        if (eventOrMessage instanceof ErrorEvent) {
            log.debug("Unhandled error event caught: ", eventOrMessage);
            eventOrMessage.preventDefault();
            message = eventOrMessage.message;
        } else if (eventOrMessage instanceof Event) {
            log.debug(
                "Unhandled error caught, parameter was event, but not ErrorEvent: ",
                eventOrMessage,
            );
            eventOrMessage.preventDefault();
        } else {
            message = String(eventOrMessage);
            log.error("Unhandled error caught: ", message, source, lineno, colno);
        }
        message ??= "An unexpected error occurred.";

        if (message === "ResizeObserver loop completed with undelivered notifications.") {
            log.debug(
                "Error ignored: 'ResizeObserver loop completed with undelivered notifications.'",
            );
            return;
        }

        toasts.error(message);
    }
</script>

<svelte:window onunhandledrejection={onAsyncError} onerror={onError} />

<ToastContainer />

{@render children()}
