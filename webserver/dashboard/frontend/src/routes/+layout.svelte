<script>
    import "../global.css";
    import "@fontsource-variable/open-sans";
    import "@fontsource-variable/chivo-mono";
    import Header from "$lib/components/layout/Header.svelte";
    import SideNav from "$lib/components/layout/SideNav.svelte";
    import BackdropBox from "$lib/components/ui/BackdropBox.svelte";
    import SideNavButton from "$lib/components/layout/SideNavButton.svelte";
    import { LayoutDashboard, Settings } from "@lucide/svelte";
    import { MetricsProvider } from "$lib/providers/metrics.svelte";
    import { onMount, setContext } from "svelte";

    let { children } = $props();

    setContext("metrics", MetricsProvider.createAndRefresh(fetch));
</script>

<div class="layout-grid">
    <div class="header-area">
        <Header />
    </div>
    <div class="sidenav-area">
        <SideNav>
            <SideNavButton url="/"><LayoutDashboard />Dashboard</SideNavButton>
            <SideNavButton url="/settings"><Settings />Settings</SideNavButton>
        </SideNav>
    </div>
    <div class="main-area">
        <BackdropBox --box-border-radius="25px 0px 0px 0px">
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
    }

    .sidenav-area {
        grid-area: sidenav;
    }

    .main-area {
        grid-area: main;
    }

    .page-container {
        overflow-y: auto;
        padding: 2rem;
    }
</style>
