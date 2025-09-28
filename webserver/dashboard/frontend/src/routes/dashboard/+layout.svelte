<script lang="ts">
    import Header from "$lib/components/layout/Header.svelte";
    import SideNav from "$lib/components/layout/SideNav.svelte";
    import BackdropBox from "$lib/components/ui/BackdropBox.svelte";
    import SideNavButton from "$lib/components/layout/SideNavButton.svelte";
    import { LayoutDashboard, Logs, Settings } from "@lucide/svelte";
    import { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { setContext } from "svelte";
    import { SettingsProvider } from "$lib/providers/settings/settings-provider.svelte";
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

<svelte:window onunhandledrejection={onAsyncError} />

<div class="layout-grid">
    <div class="header-area">
        <Header />
    </div>
    <div class="sidenav-area">
        <SideNav>
            <SideNavButton url="/dashboard"><LayoutDashboard />Dashboard</SideNavButton>
            <SideNavButton url="/dashboard/settings"><Settings />Settings</SideNavButton>
            <SideNavButton url="/dashboard/log"><Logs />Log</SideNavButton>
        </SideNav>
    </div>
    <div class="main-area">
        <BackdropBox --box-border-radius="20px 0px 0px 0px">
            <div class="page-container">
                {@render children()}
            </div>
        </BackdropBox>
    </div>
</div>

<style>
    .layout-grid {
        display: grid;
        grid-template-columns: auto 1fr;
        grid-template-rows: auto 1fr;
        grid-template-areas:
            "header header"
            "sidenav main";
        gap: 0;
        height: 100%;
    }

    .header-area {
        grid-area: header;
        min-height: 0;
        min-width: 0;
        width: 100%;
    }

    .sidenav-area {
        grid-area: sidenav;
        min-height: 0;
        min-width: 0;
        height: 100%;
    }

    .main-area {
        grid-area: main;
        min-height: 0;
        min-width: 0;
        height: 100%;
    }

    .page-container {
        padding: 2rem;

        overflow-y: auto;
        height: 100%;
    }
</style>
