<script lang="ts">
    import Header from "$lib/components/layout/Header.svelte";
    import SideNav from "$lib/components/layout/SideNav.svelte";
    import BackdropBox from "$lib/components/ui/BackdropBox.svelte";
    import SideNavButton from "$lib/components/layout/SideNavButton.svelte";
    import { LayoutDashboard, Logs, Settings } from "@lucide/svelte";

    let { children } = $props();
</script>

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
